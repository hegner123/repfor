package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper: create temporary directory with test files
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "repfor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir
}

func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

func createTestFile(t *testing.T, dir, name, content string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

func readFileContent(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	return string(content)
}

// Core utility function tests

func TestContainsWholeWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		word     string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"word in sentence", "hello world", "hello", true},
		{"word at end", "say hello", "hello", true},
		{"word with punctuation", "hello, world", "hello", true},
		{"partial match should fail", "helloworld", "hello", false},
		{"substring should fail", "superhello", "hello", false},
		{"underscore is word char", "hello_world", "hello", false},
		{"space is word boundary", "hello world", "world", true},
		{"case sensitive", "Hello", "hello", false},
		{"multiple occurrences", "log logger log", "log", true},
		{"no match", "goodbye", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsWholeWord(tt.text, tt.word)
			if result != tt.expected {
				t.Errorf("containsWholeWord(%q, %q) = %v, want %v",
					tt.text, tt.word, result, tt.expected)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	wordChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	for _, ch := range wordChars {
		if !isWordChar(ch) {
			t.Errorf("isWordChar(%q) = false, want true", ch)
		}
	}

	nonWordChars := " !@#$%^&*()-+=[]{}|;:'\",.<>?/\\"
	for _, ch := range nonWordChars {
		if isWordChar(ch) {
			t.Errorf("isWordChar(%q) = true, want false", ch)
		}
	}
}

// Replacement function tests

func TestReplaceInLine(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		search          string
		replace         string
		caseInsensitive bool
		wholeWord       bool
		expected        string
	}{
		{"simple replace", "hello world", "hello", "hi", false, false, "hi world"},
		{"multiple occurrences", "test test test", "test", "exam", false, false, "exam exam exam"},
		{"no match", "hello world", "goodbye", "hi", false, false, "hello world"},
		{"case insensitive", "Hello World", "hello", "hi", true, false, "hi World"},
		{"whole word only", "log logger log", "log", "trace", false, true, "trace logger trace"},
		{"whole word no match", "logger", "log", "trace", false, true, "logger"},
		{"case insensitive whole word", "Log Logger log", "log", "trace", true, true, "trace Logger trace"},
		{"partial replace", "password", "word", "term", false, false, "passterm"},
		{"whole word prevents partial", "password", "word", "term", false, true, "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceInLine(tt.line, tt.search, tt.replace, tt.caseInsensitive, tt.wholeWord)
			if result != tt.expected {
				t.Errorf("replaceInLine(%q, %q, %q, %v, %v) = %q, want %q",
					tt.line, tt.search, tt.replace, tt.caseInsensitive, tt.wholeWord,
					result, tt.expected)
			}
		})
	}
}

func TestCaseInsensitiveReplace(t *testing.T) {
	tests := []struct {
		line     string
		search   string
		replace  string
		expected string
	}{
		{"hello world", "hello", "hi", "hi world"},
		{"Hello World", "hello", "hi", "hi World"},
		{"HELLO world", "hello", "hi", "hi world"},
		{"hello Hello HELLO", "hello", "hi", "hi hi hi"},
		{"no match", "goodbye", "hi", "no match"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := caseInsensitiveReplace(tt.line, tt.search, tt.replace)
			if result != tt.expected {
				t.Errorf("caseInsensitiveReplace(%q, %q, %q) = %q, want %q",
					tt.line, tt.search, tt.replace, result, tt.expected)
			}
		})
	}
}

func TestWholeWordReplace(t *testing.T) {
	tests := []struct {
		line     string
		search   string
		replace  string
		expected string
	}{
		{"log logger log", "log", "trace", "trace logger trace"},
		{"logger", "log", "trace", "logger"},
		{"log", "log", "trace", "trace"},
		{"_log_", "log", "trace", "_log_"},
		{"log_file", "log", "trace", "log_file"},
		{"file_log", "log", "trace", "file_log"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := wholeWordReplace(tt.line, tt.search, tt.replace)
			if result != tt.expected {
				t.Errorf("wholeWordReplace(%q, %q, %q) = %q, want %q",
					tt.line, tt.search, tt.replace, result, tt.expected)
			}
		})
	}
}

func TestCountReplacements(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		search          string
		caseInsensitive bool
		wholeWord       bool
		expected        int
	}{
		{"simple count", "test test test", "test", false, false, 3},
		{"no matches", "hello world", "test", false, false, 0},
		{"case insensitive", "Test test TEST", "test", true, false, 3},
		{"whole word only", "log logger log", "log", false, true, 2},
		{"case insensitive whole word", "Log logger log", "log", true, true, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countReplacements(tt.line, tt.search, tt.caseInsensitive, tt.wholeWord)
			if result != tt.expected {
				t.Errorf("countReplacements(%q, %q, %v, %v) = %d, want %d",
					tt.line, tt.search, tt.caseInsensitive, tt.wholeWord, result, tt.expected)
			}
		})
	}
}

// File operation tests

func TestReplaceInFile_DryRun(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "hello world\nhello again\ngoodbye world\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "hello",
		Replace: "hi",
		DryRun:  true,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 2 {
		t.Errorf("Expected 2 lines changed, got %d", linesChanged)
	}

	if replacements != 2 {
		t.Errorf("Expected 2 replacements, got %d", replacements)
	}

	// Verify file was NOT modified
	actualContent := readFileContent(t, filePath)
	if actualContent != content {
		t.Errorf("File was modified in dry-run mode")
	}
}

func TestReplaceInFile_ActualReplace(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "hello world\nhello again\ngoodbye world\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "hello",
		Replace: "hi",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 2 {
		t.Errorf("Expected 2 lines changed, got %d", linesChanged)
	}

	if replacements != 2 {
		t.Errorf("Expected 2 replacements, got %d", replacements)
	}

	// Verify file was modified
	actualContent := readFileContent(t, filePath)
	expectedContent := "hi world\nhi again\ngoodbye world\n"
	if actualContent != expectedContent {
		t.Errorf("File content incorrect.\nExpected:\n%s\nGot:\n%s", expectedContent, actualContent)
	}
}

func TestReplaceInFile_WithExclude(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "result = calculate()\ndirResult = process()\nreturn result\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "result",
		Replace: "res",
		Exclude: []string{"dirResult"},
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 2 {
		t.Errorf("Expected 2 lines changed, got %d", linesChanged)
	}

	if replacements != 2 {
		t.Errorf("Expected 2 replacements, got %d", replacements)
	}

	// Verify dirResult line was excluded
	actualContent := readFileContent(t, filePath)
	if strings.Contains(actualContent, "dirResult") == false {
		t.Errorf("dirResult should not have been replaced")
	}

	expectedContent := "res = calculate()\ndirResult = process()\nreturn res\n"
	if actualContent != expectedContent {
		t.Errorf("File content incorrect.\nExpected:\n%s\nGot:\n%s", expectedContent, actualContent)
	}
}

func TestReplaceInFile_CaseInsensitive(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "Error occurred\nerror message\nERROR code\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:          "error",
		Replace:         "failure",
		CaseInsensitive: true,
		DryRun:          false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 3 {
		t.Errorf("Expected 3 lines changed, got %d", linesChanged)
	}

	if replacements != 3 {
		t.Errorf("Expected 3 replacements, got %d", replacements)
	}

	actualContent := readFileContent(t, filePath)
	expectedContent := "failure occurred\nfailure message\nfailure code\n"
	if actualContent != expectedContent {
		t.Errorf("File content incorrect.\nExpected:\n%s\nGot:\n%s", expectedContent, actualContent)
	}
}

func TestReplaceInFile_WholeWord(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "log message\nlogger created\nlog\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:    "log",
		Replace:   "trace",
		WholeWord: true,
		DryRun:    false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 2 {
		t.Errorf("Expected 2 lines changed, got %d", linesChanged)
	}

	if replacements != 2 {
		t.Errorf("Expected 2 replacements, got %d", replacements)
	}

	actualContent := readFileContent(t, filePath)
	expectedContent := "trace message\nlogger created\ntrace\n"
	if actualContent != expectedContent {
		t.Errorf("File content incorrect.\nExpected:\n%s\nGot:\n%s", expectedContent, actualContent)
	}
}

// Directory operation tests

func TestReplaceInDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	createTestFile(t, tmpDir, "test1.txt", "hello world\n")
	createTestFile(t, tmpDir, "test2.txt", "hello again\n")
	createTestFile(t, tmpDir, "test3.go", "hello go\n")

	config := Config{
		Search:  "hello",
		Replace: "hi",
		Ext:     ".txt",
		DryRun:  false,
	}

	result, err := replaceInDirectory(tmpDir, config)
	if err != nil {
		t.Fatalf("replaceInDirectory failed: %v", err)
	}

	if result.FilesModified != 2 {
		t.Errorf("Expected 2 files modified, got %d", result.FilesModified)
	}

	if result.LinesChanged != 2 {
		t.Errorf("Expected 2 lines changed, got %d", result.LinesChanged)
	}

	if result.TotalReplacements != 2 {
		t.Errorf("Expected 2 total replacements, got %d", result.TotalReplacements)
	}

	// Verify .go file was not modified
	goContent := readFileContent(t, filepath.Join(tmpDir, "test3.go"))
	if !strings.Contains(goContent, "hello") {
		t.Errorf(".go file should not have been modified due to extension filter")
	}
}

func TestReplaceInDirectories_MultiDir(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)

	createTestFile(t, dir1, "test1.txt", "hello world\n")
	createTestFile(t, dir2, "test2.txt", "hello again\n")

	config := Config{
		Dirs:    []string{dir1, dir2},
		Search:  "hello",
		Replace: "hi",
		DryRun:  false,
	}

	result, err := replaceInDirectories(config)
	if err != nil {
		t.Fatalf("replaceInDirectories failed: %v", err)
	}

	if len(result.Directories) != 2 {
		t.Errorf("Expected 2 directories, got %d", len(result.Directories))
	}

	totalFiles := 0
	for _, dir := range result.Directories {
		totalFiles += dir.FilesModified
	}

	if totalFiles != 2 {
		t.Errorf("Expected 2 total files modified, got %d", totalFiles)
	}
}

// Integration tests

func TestWriteFile_PreservesLineEndings(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	lines := []string{"line1", "line2", "line3"}
	filePath := filepath.Join(tmpDir, "test.txt")

	err := writeFile(filePath, lines)
	if err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	content := readFileContent(t, filePath)
	expected := "line1\nline2\nline3\n"
	if content != expected {
		t.Errorf("File content incorrect.\nExpected:\n%q\nGot:\n%q", expected, content)
	}
}

func TestReplaceInFile_EmptyFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	filePath := createTestFile(t, tmpDir, "empty.txt", "")

	config := Config{
		Search:  "hello",
		Replace: "hi",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 0 {
		t.Errorf("Expected 0 lines changed, got %d", linesChanged)
	}

	if replacements != 0 {
		t.Errorf("Expected 0 replacements, got %d", replacements)
	}
}

func TestReplaceInFile_NoMatches(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "goodbye world\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "hello",
		Replace: "hi",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	if linesChanged != 0 {
		t.Errorf("Expected 0 lines changed, got %d", linesChanged)
	}

	if replacements != 0 {
		t.Errorf("Expected 0 replacements, got %d", replacements)
	}

	// Verify file unchanged
	actualContent := readFileContent(t, filePath)
	if actualContent != content {
		t.Errorf("File should not have been modified")
	}
}
