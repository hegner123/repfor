package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Concurrency and Race Condition Tests

func TestReplaceInFile_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create multiple files
	numFiles := 100
	filePaths := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf("Line 1 contains target\nLine 2 has target\nLine 3 target here\n")
		filePaths[i] = createTestFile(t, tmpDir, fmt.Sprintf("file%03d.txt", i), content)
	}

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// Process files concurrently
	var wg sync.WaitGroup
	var errors atomic.Int32
	var totalLines atomic.Int32
	var totalReplacements atomic.Int32

	for _, path := range filePaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			lines, reps, err := replaceInFile(p, config)
			if err != nil {
				errors.Add(1)
				t.Errorf("replaceInFile failed: %v", err)
				return
			}
			totalLines.Add(int32(lines))
			totalReplacements.Add(int32(reps))
		}(path)
	}

	wg.Wait()

	if errors.Load() > 0 {
		t.Fatalf("Had %d errors during concurrent processing", errors.Load())
	}

	expectedLines := int32(numFiles * 3) // 3 lines per file
	if totalLines.Load() != expectedLines {
		t.Errorf("Expected %d total lines changed, got %d", expectedLines, totalLines.Load())
	}
}

func TestReplaceInDirectory_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create files
	for i := 0; i < 50; i++ {
		content := fmt.Sprintf("file %d target content\n", i)
		createTestFile(t, tmpDir, fmt.Sprintf("file%03d.txt", i), content)
	}

	// Multiple goroutines trying to process the same directory
	var wg sync.WaitGroup
	numGoroutines := 5
	results := make(chan *DirectoryResult, numGoroutines)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := replaceInDirectory(tmpDir, config)
			if err != nil {
				t.Errorf("replaceInDirectory failed: %v", err)
				return
			}
			results <- result
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var resultCount int
	for range results {
		resultCount++
	}

	if resultCount != numGoroutines {
		t.Errorf("Expected %d results, got %d", numGoroutines, resultCount)
	}
}

func TestReplaceInDirectories_ParallelDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create multiple directories
	numDirs := 10
	dirs := make([]string, numDirs)
	for i := 0; i < numDirs; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir%03d", i))
		os.Mkdir(dir, 0755)
		dirs[i] = dir

		// Create files in each directory
		for j := 0; j < 20; j++ {
			content := "target content\n"
			createTestFile(t, dir, fmt.Sprintf("file%03d.txt", j), content)
		}
	}

	config := Config{
		Dirs:    dirs,
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// Process directories
	result, err := replaceInDirectories(config)
	if err != nil {
		t.Fatalf("replaceInDirectories failed: %v", err)
	}

	if len(result.Directories) != numDirs {
		t.Errorf("Expected %d directories, got %d", numDirs, len(result.Directories))
	}

	totalFiles := 0
	for _, dir := range result.Directories {
		totalFiles += dir.FilesModified
	}

	expectedFiles := numDirs * 20
	if totalFiles != expectedFiles {
		t.Errorf("Expected %d total files, got %d", expectedFiles, totalFiles)
	}
}

// Race Condition Tests

func TestCaseInsensitiveReplace_RaceCondition(t *testing.T) {
	// Test for race conditions in case-insensitive replacement
	line := "Hello HELLO hello HeLLo"
	search := "hello"
	replace := "hi"

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := caseInsensitiveReplace(line, search, replace)
			expected := "hi hi hi hi"
			if result != expected {
				t.Errorf("Race condition detected: got %q, want %q", result, expected)
			}
		}()
	}

	wg.Wait()
}

func TestWholeWordReplace_RaceCondition(t *testing.T) {
	line := "log logger log logging log"
	search := "log"
	replace := "trace"

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := wholeWordReplace(line, search, replace)
			expected := "trace logger trace logging trace"
			if result != expected {
				t.Errorf("Race condition detected: got %q, want %q", result, expected)
			}
		}()
	}

	wg.Wait()
}

func TestContainsWholeWord_RaceCondition(t *testing.T) {
	text := "hello world test"
	word := "world"

	var wg sync.WaitGroup
	numGoroutines := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := containsWholeWord(text, word)
			if !result {
				t.Error("Race condition detected: expected true")
			}
		}()
	}

	wg.Wait()
}

// Stress Tests

func TestReplaceInFile_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create a very large file with many replacements
	numLines := 100000
	lines := make([]string, numLines)
	for i := 0; i < numLines; i++ {
		if i%10 == 0 {
			lines[i] = "target target target target target"
		} else {
			lines[i] = "normal line content"
		}
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	filePath := createTestFile(t, tmpDir, "stress.txt", content)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	start := time.Now()
	linesChanged, replacements, err := replaceInFile(filePath, config)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	expectedLines := numLines / 10
	if linesChanged != expectedLines {
		t.Errorf("Expected %d lines changed, got %d", expectedLines, linesChanged)
	}

	expectedReplacements := expectedLines * 5 // 5 per line
	if replacements != expectedReplacements {
		t.Errorf("Expected %d replacements, got %d", expectedReplacements, replacements)
	}

	t.Logf("Processed %d lines with %d replacements in %v", numLines, replacements, duration)

	// Performance check: should complete in reasonable time
	if duration > 30*time.Second {
		t.Errorf("Processing took too long: %v", duration)
	}
}

func TestReplaceInDirectory_StressManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create many files
	numFiles := 1000
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf("File %d target content\n", i)
		createTestFile(t, tmpDir, fmt.Sprintf("file%04d.txt", i), content)
	}

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	start := time.Now()
	result, err := replaceInDirectory(tmpDir, config)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("replaceInDirectory failed: %v", err)
	}

	if result.FilesModified != numFiles {
		t.Errorf("Expected %d files modified, got %d", numFiles, result.FilesModified)
	}

	t.Logf("Processed %d files in %v", numFiles, duration)

	if duration > 60*time.Second {
		t.Errorf("Processing took too long: %v", duration)
	}
}

// Memory Stress Tests

func TestReplaceInFile_MemoryStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stress test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Monitor memory before
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Create large file
	numLines := 500000
	lines := make([]string, numLines)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d with some target content here", i)
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	filePath := createTestFile(t, tmpDir, "large.txt", content)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	_, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	// Force GC and check memory
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memIncrease := memAfter.Alloc - memBefore.Alloc
	t.Logf("Memory increase: %d bytes (%.2f MB)", memIncrease, float64(memIncrease)/(1024*1024))

	// Should not leak excessive memory
	maxMemIncrease := uint64(500 * 1024 * 1024) // 500 MB
	if memIncrease > maxMemIncrease {
		t.Errorf("Excessive memory usage: %d bytes", memIncrease)
	}
}

// Goroutine Leak Tests

func TestNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	createTestFile(t, tmpDir, "test.txt", "target content\n")

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	numBefore := runtime.NumGoroutine()

	// Run replacements many times
	for i := 0; i < 100; i++ {
		_, _, err := replaceInFile(filepath.Join(tmpDir, "test.txt"), config)
		if err != nil {
			t.Fatalf("replaceInFile failed: %v", err)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	numAfter := runtime.NumGoroutine()

	// Allow small variance
	if numAfter > numBefore+5 {
		t.Errorf("Potential goroutine leak: before=%d, after=%d", numBefore, numAfter)
	}
}

// Concurrent Dry-Run Tests

func TestDryRun_ConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent dry-run test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	originalContent := "target target target\n"
	filePath := createTestFile(t, tmpDir, "test.txt", originalContent)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  true,
	}

	// Multiple concurrent dry-runs
	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := replaceInFile(filePath, config)
			if err != nil {
				t.Errorf("replaceInFile failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify file unchanged
	actualContent := readFileContent(t, filePath)
	if actualContent != originalContent {
		t.Error("File was modified during dry-run")
	}
}

// Timeout and Cancellation Tests

func TestReplaceInFile_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create extremely large file
	numLines := 1000000
	lines := make([]string, numLines)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d content", i)
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	filePath := createTestFile(t, tmpDir, "huge.txt", content)

	config := Config{
		Search:  "content",
		Replace: "data",
		DryRun:  false,
	}

	done := make(chan bool)
	var err error

	go func() {
		_, _, err = replaceInFile(filePath, config)
		done <- true
	}()

	select {
	case <-done:
		if err != nil {
			t.Fatalf("replaceInFile failed: %v", err)
		}
		t.Log("Long-running operation completed successfully")
	case <-time.After(2 * time.Minute):
		t.Fatal("Operation timed out after 2 minutes")
	}
}

// Concurrent Directory Scanning

func TestReplaceInDirectories_ConcurrentDirs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent directory test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create multiple directories with files
	numDirs := 20
	dirs := make([]string, numDirs)
	for i := 0; i < numDirs; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir%02d", i))
		os.Mkdir(dir, 0755)
		dirs[i] = dir

		for j := 0; j < 50; j++ {
			content := "target content\n"
			createTestFile(t, dir, fmt.Sprintf("file%02d.txt", j), content)
		}
	}

	config := Config{
		Dirs:    dirs,
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	start := time.Now()
	result, err := replaceInDirectories(config)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("replaceInDirectories failed: %v", err)
	}

	totalFiles := 0
	for _, dir := range result.Directories {
		totalFiles += dir.FilesModified
	}

	expectedFiles := numDirs * 50
	if totalFiles != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, totalFiles)
	}

	t.Logf("Processed %d directories (%d files) in %v", numDirs, totalFiles, duration)
}
