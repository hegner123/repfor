//go:build stress

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Stress Tests — guarded by build tag, not compiled in CI.
// Run locally with: go test -tags stress -v -timeout 5m

func TestReplaceInFile_StressTest(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

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

	expectedReplacements := expectedLines * 5
	if replacements != expectedReplacements {
		t.Errorf("Expected %d replacements, got %d", expectedReplacements, replacements)
	}

	t.Logf("Processed %d lines with %d replacements in %v", numLines, replacements, duration)

	if duration > 30*time.Second {
		t.Errorf("Processing took too long: %v", duration)
	}
}

func TestReplaceInDirectory_StressManyFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

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

func TestReplaceInFile_MemoryStress(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

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

	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memIncrease := memAfter.Alloc - memBefore.Alloc
	t.Logf("Memory increase: %d bytes (%.2f MB)", memIncrease, float64(memIncrease)/(1024*1024))

	maxMemIncrease := uint64(500 * 1024 * 1024)
	if memIncrease > maxMemIncrease {
		t.Errorf("Excessive memory usage: %d bytes", memIncrease)
	}
}

func TestNoGoroutineLeaks(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	createTestFile(t, tmpDir, "test.txt", "target content\n")

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	numBefore := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		_, _, err := replaceInFile(filepath.Join(tmpDir, "test.txt"), config)
		if err != nil {
			t.Fatalf("replaceInFile failed: %v", err)
		}
	}

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	numAfter := runtime.NumGoroutine()

	if numAfter > numBefore+5 {
		t.Errorf("Potential goroutine leak: before=%d, after=%d", numBefore, numAfter)
	}
}

func TestDryRun_ConcurrentSafety(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	originalContent := "target target target\n"
	filePath := createTestFile(t, tmpDir, "test.txt", originalContent)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  true,
	}

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

	actualContent := readFileContent(t, filePath)
	if actualContent != originalContent {
		t.Error("File was modified during dry-run")
	}
}

func TestReplaceInFile_LongRunning(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

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

func TestReplaceInDirectories_ConcurrentDirs(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	numDirs := 20
	dirs := make([]string, numDirs)
	for i := 0; i < numDirs; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir%02d", i))
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
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

// Heavy tests from main_advanced_test.go

func TestReplaceInFile_LargeFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	lines := make([]string, 100000)
	for i := range lines {
		if i%1000 == 0 {
			lines[i] = fmt.Sprintf("Line %d contains target text", i)
		} else {
			lines[i] = fmt.Sprintf("Line %d regular content", i)
		}
	}
	content := strings.Join(lines, "\n") + "\n"
	filePath := createTestFile(t, tmpDir, "large.txt", content)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	expectedLines := 100
	if linesChanged != expectedLines {
		t.Errorf("Expected %d lines changed, got %d", expectedLines, linesChanged)
	}

	if replacements != expectedLines {
		t.Errorf("Expected %d replacements, got %d", expectedLines, replacements)
	}
}

func TestReplaceInFile_VeryLongLines(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	longLine := strings.Repeat("a", 1000000) + "target" + strings.Repeat("b", 1000000)
	content := longLine + "\n"
	filePath := createTestFile(t, tmpDir, "longlines.txt", content)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 1 {
		t.Errorf("Expected 1 line changed, got %d", linesChanged)
	}

	if replacements != 1 {
		t.Errorf("Expected 1 replacement, got %d", replacements)
	}

	actualContent := readFileContent(t, filePath)
	if !strings.Contains(actualContent, "REPLACED") {
		t.Error("REPLACED not found in file")
	}
	if strings.Contains(actualContent, "target") {
		t.Error("target still found in file, replacement failed")
	}
}

func TestReplaceInFile_ManySmallFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	numFiles := 1000
	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf("File %d contains target\n", i)
		createTestFile(t, tmpDir, fmt.Sprintf("file%04d.txt", i), content)
	}

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	result, err := replaceInDirectory(tmpDir, config)
	if err != nil {
		t.Fatalf("replaceInDirectory failed: %v", err)
	}

	if result.FilesModified != numFiles {
		t.Errorf("Expected %d files modified, got %d", numFiles, result.FilesModified)
	}
}

func TestCountReplacements_ManyOccurrences(t *testing.T) {
	line := strings.Repeat("x ", 50000)
	count := countReplacements(line, "x", false, false)

	expected := 50000
	if count != expected {
		t.Errorf("Expected %d replacements, got %d", expected, count)
	}
}
