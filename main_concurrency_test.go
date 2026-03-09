package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
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
		content := "Line 1 contains target\nLine 2 has target\nLine 3 target here\n"
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
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
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
