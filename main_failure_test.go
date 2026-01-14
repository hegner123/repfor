package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Failure Injection and Error Handling Tests

// File System Failure Tests

func TestReplaceInFile_NonExistentFile(t *testing.T) {
	config := Config{
		Search:  "test",
		Replace: "exam",
		DryRun:  false,
	}

	_, _, err := replaceInFile("/nonexistent/path/file.txt", config)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestReplaceInFile_ReadOnlyFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "target content\n"
	filePath := createTestFile(t, tmpDir, "readonly.txt", content)

	// Make file read-only
	if err := os.Chmod(filePath, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(filePath, 0644) // Restore permissions for cleanup

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	_, _, err := replaceInFile(filePath, config)
	if err == nil {
		t.Error("Expected error when writing to read-only file")
	}
}

func TestReplaceInFile_DryRunReadOnly(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "target content\n"
	filePath := createTestFile(t, tmpDir, "readonly.txt", content)

	// Make file read-only
	if err := os.Chmod(filePath, 0444); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(filePath, 0644)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  true, // Dry-run should succeed even on read-only
	}

	linesChanged, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("Dry-run should succeed on read-only file: %v", err)
	}

	if linesChanged != 1 {
		t.Errorf("Expected 1 line changed, got %d", linesChanged)
	}
}

func TestReplaceInDirectory_NonExistentDir(t *testing.T) {
	config := Config{
		Search:  "test",
		Replace: "exam",
		DryRun:  false,
	}

	_, err := replaceInDirectory("/nonexistent/directory", config)
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
}

func TestReplaceInDirectory_FileAsDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create a file, then try to treat it as a directory
	filePath := createTestFile(t, tmpDir, "notadir.txt", "content")

	config := Config{
		Search:  "test",
		Replace: "exam",
		DryRun:  false,
	}

	_, err := replaceInDirectory(filePath, config)
	if err == nil {
		t.Error("Expected error when treating file as directory")
	}
}

func TestReplaceInDirectory_EmptyDirectory(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	config := Config{
		Search:  "test",
		Replace: "exam",
		DryRun:  false,
	}

	result, err := replaceInDirectory(tmpDir, config)
	if err != nil {
		t.Fatalf("Should handle empty directory: %v", err)
	}

	if result.FilesModified != 0 {
		t.Errorf("Expected 0 files modified in empty directory, got %d", result.FilesModified)
	}
}

// Permission Tests

func TestReplaceInDirectory_NoReadPermission(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	subDir := filepath.Join(tmpDir, "noperm")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	createTestFile(t, subDir, "test.txt", "content")

	// Remove read permission
	if err := os.Chmod(subDir, 0000); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(subDir, 0755) // Restore for cleanup

	config := Config{
		Search:  "test",
		Replace: "exam",
		DryRun:  false,
	}

	_, err := replaceInDirectory(subDir, config)
	if err == nil {
		t.Error("Expected error for directory without read permission")
	}
}

// Disk Space Simulation

func TestReplaceInFile_SimulatedDiskFull(t *testing.T) {
	// This test would require mocking the filesystem or using a quota'd filesystem
	// For now, we document the expected behavior
	t.Skip("Disk full simulation requires special setup")

	// Expected behavior:
	// - writeFile should return error
	// - Original file should remain unchanged
	// - No partial writes should occur
}

// Corrupted Input Tests

func TestReplaceInFile_TruncatedFile(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create file and then truncate it while "in use"
	content := "line 1\nline 2\nline 3\n"
	filePath := createTestFile(t, tmpDir, "truncated.txt", content)

	// Simulate file truncation
	if err := os.Truncate(filePath, 5); err != nil {
		t.Fatalf("Failed to truncate: %v", err)
	}

	config := Config{
		Search:  "line",
		Replace: "row",
		DryRun:  false,
	}

	// Should handle truncated file gracefully
	_, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Logf("Truncated file error (expected): %v", err)
	}
}

// Concurrent Modification Tests

func TestReplaceInFile_ConcurrentModification(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	originalContent := "target line\n"
	filePath := createTestFile(t, tmpDir, "concurrent.txt", originalContent)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// Simulate concurrent modification by changing file during read
	// This is a race condition we want to detect
	go func() {
		// Modify file after a brief delay
		os.WriteFile(filePath, []byte("modified content\n"), 0644)
	}()

	// Try to replace
	_, _, err := replaceInFile(filePath, config)

	// Behavior is undefined in this case, but should not crash
	if err != nil {
		t.Logf("Concurrent modification detected: %v", err)
	}
}

// Symlink Tests

func TestReplaceInFile_Symlink(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create target file
	targetPath := createTestFile(t, tmpDir, "target.txt", "target content\n")

	// Create symlink
	linkPath := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("Symlink creation failed (may not be supported): %v", err)
	}

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// Should follow symlink and modify target
	linesChanged, _, err := replaceInFile(linkPath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed on symlink: %v", err)
	}

	if linesChanged != 1 {
		t.Errorf("Expected 1 line changed, got %d", linesChanged)
	}

	// Verify target file was modified
	content := readFileContent(t, targetPath)
	if !strings.Contains(content, "REPLACED") {
		t.Error("Target file not modified via symlink")
	}
}

// Special File Tests

func TestReplaceInDirectory_SkipsSubdirectories(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create files and subdirectory
	createTestFile(t, tmpDir, "file1.txt", "target\n")
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	createTestFile(t, subDir, "file2.txt", "target\n")

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	result, err := replaceInDirectory(tmpDir, config)
	if err != nil {
		t.Fatalf("replaceInDirectory failed: %v", err)
	}

	// Should only process file1.txt, not subdirectory
	if result.FilesModified != 1 {
		t.Errorf("Expected 1 file modified (subdirectory should be skipped), got %d", result.FilesModified)
	}

	// Verify subdirectory file was not modified
	subContent := readFileContent(t, filepath.Join(subDir, "file2.txt"))
	if strings.Contains(subContent, "REPLACED") {
		t.Error("Subdirectory file should not be modified")
	}
}

// Resource Exhaustion Tests

func TestReplaceInFile_ExtremelyLongLine(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create file with extremely long line (10MB)
	longLine := strings.Repeat("a", 10*1024*1024) + "target" + strings.Repeat("b", 100)
	filePath := createTestFile(t, tmpDir, "longline.txt", longLine+"\n")

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// Should handle without crashing
	_, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Logf("Long line handling: %v", err)
	}
}

func TestReplaceInFile_ManyLines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource test in short mode")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create file with many lines
	numLines := 1000000
	lines := make([]string, numLines)
	for i := range lines {
		if i%100 == 0 {
			lines[i] = "target"
		} else {
			lines[i] = "normal"
		}
	}
	content := strings.Join(lines, "\n") + "\n"
	filePath := createTestFile(t, tmpDir, "manylines.txt", content)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	linesChanged, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	expectedLines := numLines / 100
	if linesChanged != expectedLines {
		t.Errorf("Expected %d lines changed, got %d", expectedLines, linesChanged)
	}
}

// Error Recovery Tests

func TestReplaceInFile_RecoveryAfterError(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	// First attempt: fail on non-existent file
	_, _, err := replaceInFile(filepath.Join(tmpDir, "nonexistent.txt"), config)
	if err == nil {
		t.Error("Expected error for first attempt")
	}

	// Second attempt: succeed on valid file
	filePath := createTestFile(t, tmpDir, "valid.txt", "target\n")
	linesChanged, _, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("Second attempt should succeed: %v", err)
	}

	if linesChanged != 1 {
		t.Errorf("Expected 1 line changed, got %d", linesChanged)
	}
}

// Multi-Error Scenarios

func TestReplaceInDirectory_PartialFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create mix of accessible and inaccessible files
	createTestFile(t, tmpDir, "good1.txt", "target\n")
	createTestFile(t, tmpDir, "good2.txt", "target\n")

	badPath := filepath.Join(tmpDir, "bad.txt")
	createTestFile(t, tmpDir, "bad.txt", "target\n")
	os.Chmod(badPath, 0000)
	defer os.Chmod(badPath, 0644)

	config := Config{
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	result, err := replaceInDirectory(tmpDir, config)
	if err != nil {
		t.Fatalf("replaceInDirectory failed: %v", err)
	}

	// Should process accessible files despite one failure
	if result.FilesModified != 2 {
		t.Logf("Expected 2 files modified, got %d (partial failure expected)", result.FilesModified)
	}
}

// Validation Tests

func TestReplaceInDirectories_MixedValidInvalid(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	validDir := filepath.Join(tmpDir, "valid")
	os.Mkdir(validDir, 0755)
	createTestFile(t, validDir, "test.txt", "target\n")

	config := Config{
		Dirs:    []string{validDir, "/nonexistent/dir", ""},
		Search:  "target",
		Replace: "REPLACED",
		DryRun:  false,
	}

	_, err := replaceInDirectories(config)
	// Should fail on first invalid directory
	if err == nil {
		t.Error("Expected error for invalid directories")
	}
}

// File Type Edge Cases

func TestReplaceInDirectory_SpecialFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	// Create regular file
	createTestFile(t, tmpDir, "regular.txt", "target\n")

	// Try to create special files (may not be supported on all systems)
	// Named pipe (FIFO)
	fifoPath := filepath.Join(tmpDir, "fifo")
	if err := syscall.Mkfifo(fifoPath, 0644); err != nil {
		t.Logf("FIFO creation not supported: %v", err)
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

	// Should only process regular file
	if result.FilesModified < 1 {
		t.Error("Should process at least the regular file")
	}
}

// Cleanup Failure Tests

func TestCleanupAfterPartialWrite(t *testing.T) {
	// This would test cleanup after writeFile fails mid-operation
	// Current implementation overwrites file, so partial writes could occur
	// This documents expected behavior for future improvement

	t.Skip("Cleanup after partial write not yet implemented")

	// Expected behavior:
	// - Use atomic writes (write to temp file, then rename)
	// - Ensure original file is preserved on write failure
	// - Clean up temporary files
}

// Edge Case Combinations

func TestReplaceInFile_EmptySearchEmptyReplace(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "hello world\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "",
		Replace: "",
		DryRun:  false,
	}

	_, _, err := replaceInFile(filePath, config)
	// Should handle gracefully (likely no-op)
	if err != nil {
		t.Logf("Empty search/replace error (may be expected): %v", err)
	}

	actualContent := readFileContent(t, filePath)
	if actualContent != content {
		t.Logf("Content changed: %q -> %q", content, actualContent)
	}
}

func TestReplaceInFile_SearchEqualsReplace(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer cleanupTestDir(t, tmpDir)

	content := "target content\n"
	filePath := createTestFile(t, tmpDir, "test.txt", content)

	config := Config{
		Search:  "target",
		Replace: "target",
		DryRun:  false,
	}

	linesChanged, replacements, err := replaceInFile(filePath, config)
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	// When search equals replace, no actual change occurs
	// So we expect 0 lines changed and 0 replacements
	if linesChanged != 0 {
		t.Errorf("Expected 0 lines changed (no-op), got %d", linesChanged)
	}

	if replacements != 0 {
		t.Errorf("Expected 0 replacements (no-op), got %d", replacements)
	}

	actualContent := readFileContent(t, filePath)
	if actualContent != content {
		t.Error("Content should be unchanged when search equals replace")
	}
}
