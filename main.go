package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type FileModification struct {
	Path         string `json:"path"`
	LinesChanged int    `json:"lines_changed"`
	Replacements int    `json:"replacements"`
}

type DirectoryResult struct {
	Dir               string             `json:"dir"`
	FilesModified     int                `json:"files_modified"`
	LinesChanged      int                `json:"lines_changed"`
	TotalReplacements int                `json:"total_replacements"`
	Files             []FileModification `json:"files"`
}

type Result struct {
	Summary     string            `json:"summary"`
	Directories []DirectoryResult `json:"directories"`
	DryRun      bool              `json:"dry_run,omitempty"`
}

type Config struct {
	Dirs            []string
	Files           []string // file mode (takes precedence over Dirs)
	Search          string
	Replace         string
	Ext             string
	Exclude         []string
	CaseInsensitive bool
	WholeWord       bool
	DryRun          bool
	Recursive       bool
	CLIMode         bool
	Verbose         bool
	ReplaceSet      bool // tracks if --replace was explicitly provided (allows empty string)
}

// MCP JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools map[string]bool `json:"tools"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
}

type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolCallResult struct {
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	config := parseFlags()

	if config.CLIMode {
		runCLI(config)
		return
	}

	runMCPServer()
}

func parseFlags() Config {
	config := Config{}
	var dirStr string
	var excludeStr string

	flag.BoolVar(&config.CLIMode, "cli", false, "Run in CLI mode (default is MCP server mode)")
	flag.StringVar(&dirStr, "dir", "", "Comma-separated list of directories to search (defaults to current directory)")
	var fileStr string
	flag.StringVar(&fileStr, "file", "", "Comma-separated list of files to process (takes precedence over --dir)")
	flag.StringVar(&config.Search, "search", "", "String to search for (required)")
	flag.StringVar(&config.Replace, "replace", "", "String to replace with (required, use empty string to delete)")
	flag.StringVar(&config.Ext, "ext", "", "File extension to filter (e.g., .go, .txt)")
	flag.StringVar(&excludeStr, "exclude", "", "Comma-separated list of strings to exclude from replacement")
	flag.BoolVar(&config.CaseInsensitive, "case-insensitive", false, "Perform case-insensitive search")
	flag.BoolVar(&config.WholeWord, "whole-word", false, "Match whole words only")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview changes without modifying files")
	flag.BoolVar(&config.Recursive, "recursive", false, "Recursively search subdirectories")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show progress on stderr")

	flag.Parse()

	// Check if --replace was explicitly set (allows empty string for delete mode)
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "replace" {
			config.ReplaceSet = true
		}
	})

	if fileStr != "" {
		config.Files = strings.Split(fileStr, ",")
		for i := range config.Files {
			config.Files[i] = strings.TrimSpace(config.Files[i])
		}
	}

	if dirStr != "" {
		config.Dirs = strings.Split(dirStr, ",")
		for i := range config.Dirs {
			config.Dirs[i] = strings.TrimSpace(config.Dirs[i])
		}
	} else {
		config.Dirs = []string{"."}
	}

	if excludeStr != "" {
		config.Exclude = strings.Split(excludeStr, ",")
		for i := range config.Exclude {
			config.Exclude[i] = strings.TrimSpace(config.Exclude[i])
		}
	}

	return config
}

// Exit codes for CLI mode
const (
	ExitSuccess   = 0 // Success with changes
	ExitError     = 1 // Error occurred
	ExitNoChanges = 2 // Success but no matches found
)

func runCLI(config Config) {
	if config.Search == "" {
		fmt.Fprintln(os.Stderr, "Error: --search is required")
		flag.Usage()
		os.Exit(ExitError)
	}

	if !config.ReplaceSet {
		fmt.Fprintln(os.Stderr, "Error: --replace is required (use empty string to delete matches)")
		flag.Usage()
		os.Exit(ExitError)
	}

	// Warn if search equals replace (no-op)
	if config.Search == config.Replace {
		fmt.Fprintln(os.Stderr, "Warning: search and replace are identical, no changes will be made")
	}

	result, err := replaceInDirectories(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}

	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(ExitError)
	}

	fmt.Println(string(output))

	// Exit with appropriate code
	totalReplacements := 0
	for _, dir := range result.Directories {
		totalReplacements += dir.TotalReplacements
	}
	if totalReplacements == 0 {
		os.Exit(ExitNoChanges)
	}
}

func runMCPServer() {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "Received shutdown signal, exiting gracefully...")
		cancel()
	}()

	scanner := bufio.NewScanner(os.Stdin)

	// Channel to receive scan results
	lineChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		for scanner.Scan() {
			lineChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
		close(lineChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errChan:
			fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
			return
		case line, ok := <-lineChan:
			if !ok {
				return // stdin closed
			}
			if line == "" {
				continue
			}

			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				sendError(nil, -32700, "Parse error")
				continue
			}

			handleRequest(req)
		}
	}
}

func handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		handleInitialize(req)
	case "tools/list":
		handleToolsList(req)
	case "tools/call":
		handleToolsCall(req)
	default:
		sendError(req.ID, -32601, "Method not found")
	}
}

func handleInitialize(req JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "repfor",
			Version: "1.0.0",
		},
		Capabilities: Capabilities{
			Tools: map[string]bool{
				"list": true,
				"call": true,
			},
		},
	}
	sendResponse(req.ID, result)
}

func handleToolsList(req JSONRPCRequest) {
	result := ToolsListResult{
		Tools: []Tool{
			{
				Name:        "repfor",
				Description: "Search and replace strings in files across directories. By default scans single-depth (non-recursive), but supports recursive scanning with the 'recursive' option. In-place file modifications with extension filtering, case-insensitive search, whole-word matching, and exclude filters.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"file": {
							Type:        "array",
							Description: "Array of file paths to process. Can also accept a single string. Takes precedence over 'dir' if both are provided.",
						},
						"dir": {
							Type:        "array",
							Description: "Array of directory paths to search. Can also accept a single string for backwards compatibility. Defaults to current directory if not provided.",
						},
						"search": {
							Type:        "string",
							Description: "String to search for. Use \\n in the string to match literal newlines for multi-line patterns.",
						},
						"replace": {
							Type:        "string",
							Description: "String to replace matches with. Use \\n in the string to insert literal newlines for multi-line replacements.",
						},
						"ext": {
							Type:        "string",
							Description: "File extension to filter (e.g., '.go', '.txt'). Optional.",
						},
						"exclude": {
							Type:        "array",
							Description: "Array of strings to exclude from replacement. Lines containing any of these strings will not be modified. Optional.",
						},
						"case_insensitive": {
							Type:        "boolean",
							Description: "Perform case-insensitive search. Optional, defaults to false.",
							Default:     false,
						},
						"whole_word": {
							Type:        "boolean",
							Description: "Match whole words only. Optional, defaults to false.",
							Default:     false,
						},
						"dry_run": {
							Type:        "boolean",
							Description: "Preview changes without modifying files. Optional, defaults to false.",
							Default:     false,
						},
						"recursive": {
							Type:        "boolean",
							Description: "Recursively search subdirectories. Optional, defaults to false.",
							Default:     false,
						},
					},
					Required: []string{"search", "replace"},
				},
			},
		},
	}
	sendResponse(req.ID, result)
}

func handleToolsCall(req JSONRPCRequest) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid params")
		return
	}

	if params.Name != "repfor" {
		sendError(req.ID, -32602, "Unknown tool")
		return
	}

	search, ok := params.Arguments["search"].(string)
	if !ok {
		sendError(req.ID, -32602, "Missing or invalid 'search' parameter")
		return
	}

	replace, ok := params.Arguments["replace"].(string)
	if !ok {
		sendError(req.ID, -32602, "Missing or invalid 'replace' parameter")
		return
	}

	config := Config{
		Search:  search,
		Replace: replace,
	}

	// File mode takes precedence over directory mode
	if fileParam, exists := params.Arguments["file"]; exists {
		switch v := fileParam.(type) {
		case string:
			if v != "" {
				config.Files = []string{v}
			}
		case []any:
			config.Files = make([]string, 0, len(v))
			for _, f := range v {
				if str, ok := f.(string); ok {
					config.Files = append(config.Files, str)
				}
			}
		}
	}

	if dirParam, exists := params.Arguments["dir"]; exists {
		switch v := dirParam.(type) {
		case string:
			config.Dirs = []string{v}
		case []any:
			config.Dirs = make([]string, 0, len(v))
			for _, d := range v {
				if str, ok := d.(string); ok {
					config.Dirs = append(config.Dirs, str)
				}
			}
		}
	}

	if len(config.Dirs) == 0 {
		config.Dirs = []string{"."}
	}

	if ext, ok := params.Arguments["ext"].(string); ok {
		config.Ext = ext
	}

	if excludeArray, ok := params.Arguments["exclude"].([]any); ok {
		config.Exclude = make([]string, 0, len(excludeArray))
		for _, v := range excludeArray {
			if str, ok := v.(string); ok {
				config.Exclude = append(config.Exclude, str)
			}
		}
	}

	if caseInsensitive, ok := params.Arguments["case_insensitive"].(bool); ok {
		config.CaseInsensitive = caseInsensitive
	}

	if wholeWord, ok := params.Arguments["whole_word"].(bool); ok {
		config.WholeWord = wholeWord
	}

	if dryRun, ok := params.Arguments["dry_run"].(bool); ok {
		config.DryRun = dryRun
	}

	if recursive, ok := params.Arguments["recursive"].(bool); ok {
		config.Recursive = recursive
	}

	result, err := replaceInDirectories(config)
	if err != nil {
		sendError(req.ID, -32603, fmt.Sprintf("Replacement failed: %v", err))
		return
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		sendError(req.ID, -32603, "Failed to marshal result")
		return
	}

	response := ToolCallResult{
		Content: []ContentItem{
			{
				Type: "text",
				Text: string(jsonResult),
			},
		},
	}

	sendResponse(req.ID, response)
}

func sendResponse(id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func sendError(id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal error response: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func replaceInDirectories(config Config) (*Result, error) {
	result := &Result{
		Directories: make([]DirectoryResult, 0, len(config.Dirs)),
		DryRun:      config.DryRun,
	}

	// File mode takes precedence over directory mode
	if len(config.Files) > 0 {
		dirResult, err := replaceInFiles(config.Files, config)
		if err != nil {
			return nil, err
		}
		result.Directories = append(result.Directories, *dirResult)
	} else {
		// Collect all directories to process
		dirsToProcess := config.Dirs
		if config.Recursive {
			dirsToProcess = collectDirectoriesRecursive(config.Dirs)
		}

		for _, dir := range dirsToProcess {
			dirResult, err := replaceInDirectory(dir, config)
			if err != nil {
				return nil, err
			}
			result.Directories = append(result.Directories, *dirResult)
		}
	}

	// Generate summary
	totalFiles := 0
	totalLines := 0
	totalReplacements := 0
	dirsWithChanges := 0

	for _, dirResult := range result.Directories {
		totalFiles += dirResult.FilesModified
		totalLines += dirResult.LinesChanged
		totalReplacements += dirResult.TotalReplacements
		if dirResult.FilesModified > 0 {
			dirsWithChanges++
		}
	}

	// Build summary string
	var action string
	if config.DryRun {
		action = "Would modify"
	} else {
		action = "Modified"
	}

	fileWord := "file"
	if totalFiles != 1 {
		fileWord = "files"
	}

	lineWord := "line"
	if totalLines != 1 {
		lineWord = "lines"
	}

	replacementWord := "replacement"
	if totalReplacements != 1 {
		replacementWord = "replacements"
	}

	var dirInfo string
	if len(config.Dirs) > 1 {
		dirWord := "directory"
		if dirsWithChanges != 1 {
			dirWord = "directories"
		}
		dirInfo = fmt.Sprintf(" across %d %s", dirsWithChanges, dirWord)
	}

	result.Summary = fmt.Sprintf("%s %d %s%s: %d %s in %d %s",
		action, totalFiles, fileWord, dirInfo, totalReplacements, replacementWord, totalLines, lineWord)

	return result, nil
}

// collectDirectoriesRecursive walks the given directories and returns all directories
// including subdirectories. The input directories are included in the result.
func collectDirectoriesRecursive(dirs []string) []string {
	var allDirs []string
	seen := make(map[string]bool)

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to access %s: %v\n", path, err)
				return nil // Continue walking despite errors
			}
			if d.IsDir() {
				// Use cleaned path to avoid duplicates
				cleanPath := filepath.Clean(path)
				if !seen[cleanPath] {
					seen[cleanPath] = true
					allDirs = append(allDirs, cleanPath)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to walk directory %s: %v\n", dir, err)
		}
	}

	return allDirs
}

func replaceInDirectory(dir string, config Config) (*DirectoryResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	dirResult := &DirectoryResult{
		Dir:   dir,
		Files: make([]FileModification, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip non-regular files (FIFOs, devices, sockets, etc.)
		info, err := entry.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get file info for %s: %v\n", entry.Name(), err)
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}

		filename := entry.Name()

		if config.Ext != "" && !strings.HasSuffix(filename, config.Ext) {
			continue
		}

		fullPath := filepath.Join(dir, filename)
		linesChanged, replacements, err := replaceInFile(fullPath, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", fullPath, err)
			continue
		}

		if linesChanged > 0 {
			dirResult.Files = append(dirResult.Files, FileModification{
				Path:         filename,
				LinesChanged: linesChanged,
				Replacements: replacements,
			})
			dirResult.FilesModified++
			dirResult.LinesChanged += linesChanged
			dirResult.TotalReplacements += replacements
		}
	}

	return dirResult, nil
}

func replaceInFiles(filePaths []string, config Config) (*DirectoryResult, error) {
	dirResult := &DirectoryResult{
		Dir:   "(files)",
		Files: make([]FileModification, 0, len(filePaths)),
	}

	for _, filePath := range filePaths {
		// Verify file exists and is a regular file
		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to stat file %s: %v\n", filePath, err)
			continue
		}
		if !info.Mode().IsRegular() {
			fmt.Fprintf(os.Stderr, "Warning: not a regular file: %s\n", filePath)
			continue
		}

		// Check extension filter if specified
		if config.Ext != "" && !strings.HasSuffix(filePath, config.Ext) {
			continue
		}

		linesChanged, replacements, err := replaceInFile(filePath, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", filePath, err)
			continue
		}

		if linesChanged > 0 {
			dirResult.Files = append(dirResult.Files, FileModification{
				Path:         filePath,
				LinesChanged: linesChanged,
				Replacements: replacements,
			})
			dirResult.FilesModified++
			dirResult.LinesChanged += linesChanged
			dirResult.TotalReplacements += replacements
		}
	}

	return dirResult, nil
}

// maxLineSize is the maximum line size in bytes (10MB)
const maxLineSize = 10 * 1024 * 1024

func replaceInFile(path string, config Config) (int, int, error) {
	// Early exit: if search equals replace, it's a no-op
	if config.Search == config.Replace {
		return 0, 0, nil
	}

	// Dispatch to multiline path when search or replace contains newlines
	if isMultiline(config.Search, config.Replace) {
		return replaceInFileMultiline(path, config)
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", path, cerr)
		}
	}()

	// Detect line ending style by reading first chunk
	lineEnding := "\n" // default to Unix style
	detectBuf := make([]byte, 8192)
	n, _ := file.Read(detectBuf)
	if n > 0 {
		for i := 0; i < n-1; i++ {
			if detectBuf[i] == '\r' && detectBuf[i+1] == '\n' {
				lineEnding = "\r\n"
				break
			}
			if detectBuf[i] == '\n' {
				break // Unix style confirmed
			}
		}
	}
	// Reset file to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, fmt.Errorf("failed to seek file: %w", err)
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle very long lines (default is 64KB, set to 10MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxLineSize)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		// Provide specific error for lines that are too long
		if errors.Is(err, bufio.ErrTooLong) {
			return 0, 0, fmt.Errorf("line too long (max %dMB): %w", maxLineSize/(1024*1024), err)
		}
		return 0, 0, err
	}

	linesChanged := 0
	totalReplacements := 0
	modifiedLines := make([]string, len(lines))
	copy(modifiedLines, lines)

	searchTerm := config.Search
	replaceTerm := config.Replace
	if config.CaseInsensitive {
		searchTerm = strings.ToLower(searchTerm)
	}

	for i, line := range lines {
		lineToCheck := line
		if config.CaseInsensitive {
			lineToCheck = strings.ToLower(line)
		}

		found := false
		if config.WholeWord {
			found = containsWholeWord(lineToCheck, searchTerm)
		} else {
			found = strings.Contains(lineToCheck, searchTerm)
		}

		if !found {
			continue
		}

		excluded := false
		for _, excludePattern := range config.Exclude {
			excludeToCheck := excludePattern
			lineForExclude := line
			if config.CaseInsensitive {
				excludeToCheck = strings.ToLower(excludePattern)
				lineForExclude = lineToCheck
			}
			if strings.Contains(lineForExclude, excludeToCheck) {
				excluded = true
				// DEBUG: uncomment for diagnostics
				// fmt.Fprintf(os.Stderr, "DEBUG: Line %d excluded by pattern %q: %q\n", i, excludePattern, line)
				break
			}
		}

		if excluded {
			continue
		}

		newLine := replaceInLine(line, config.Search, replaceTerm, config.CaseInsensitive, config.WholeWord)
		if newLine != line {
			modifiedLines[i] = newLine
			linesChanged++
			totalReplacements += countReplacements(line, config.Search, config.CaseInsensitive, config.WholeWord)
		}
	}

	if linesChanged > 0 && !config.DryRun {
		err := writeFileAtomic(path, modifiedLines, lineEnding)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to write file: %w", err)
		}
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Modified: %s (%d replacements in %d lines)\n", path, totalReplacements, linesChanged)
		}
	}

	return linesChanged, totalReplacements, nil
}

func replaceInLine(line, search, replace string, caseInsensitive, wholeWord bool) string {
	if search == "" {
		return line
	}

	if !caseInsensitive && !wholeWord {
		return strings.ReplaceAll(line, search, replace)
	}

	if caseInsensitive && !wholeWord {
		return caseInsensitiveReplace(line, search, replace)
	}

	if wholeWord && !caseInsensitive {
		return wholeWordReplace(line, search, replace)
	}

	return caseInsensitiveWholeWordReplace(line, search, replace)
}

func caseInsensitiveReplace(line, search, replace string) string {
	if search == "" {
		return line
	}

	searchLower := strings.ToLower(search)
	var result strings.Builder
	result.Grow(len(line))
	remaining := line

	for {
		lineLower := strings.ToLower(remaining)
		idx := strings.Index(lineLower, searchLower)
		if idx == -1 {
			result.WriteString(remaining)
			break
		}

		result.WriteString(remaining[:idx])
		result.WriteString(replace)
		remaining = remaining[idx+len(search):]
	}

	return result.String()
}

func wholeWordReplace(line, search, replace string) string {
	if search == "" {
		return line
	}

	var result strings.Builder
	result.Grow(len(line))
	remaining := line
	searchLen := len(search)

	for {
		idx := strings.Index(remaining, search)
		if idx == -1 {
			result.WriteString(remaining)
			break
		}

		beforeOk := idx == 0 || !isWordChar(rune(remaining[idx-1]))
		afterIdx := idx + searchLen
		afterOk := afterIdx >= len(remaining) || !isWordChar(rune(remaining[afterIdx]))

		if beforeOk && afterOk {
			result.WriteString(remaining[:idx])
			result.WriteString(replace)
			remaining = remaining[afterIdx:]
		} else {
			result.WriteString(remaining[:idx+1])
			remaining = remaining[idx+1:]
		}
	}

	return result.String()
}

func caseInsensitiveWholeWordReplace(line, search, replace string) string {
	if search == "" {
		return line
	}

	var result strings.Builder
	result.Grow(len(line))
	remaining := line
	searchLower := strings.ToLower(search)
	searchLen := len(search)

	for {
		lineLower := strings.ToLower(remaining)
		idx := strings.Index(lineLower, searchLower)
		if idx == -1 {
			result.WriteString(remaining)
			break
		}

		beforeOk := idx == 0 || !isWordChar(rune(remaining[idx-1]))
		afterIdx := idx + searchLen
		afterOk := afterIdx >= len(remaining) || !isWordChar(rune(remaining[afterIdx]))

		if beforeOk && afterOk {
			result.WriteString(remaining[:idx])
			result.WriteString(replace)
			remaining = remaining[afterIdx:]
		} else {
			result.WriteString(remaining[:idx+1])
			remaining = remaining[idx+1:]
		}
	}

	return result.String()
}

func countReplacements(line, search string, caseInsensitive, wholeWord bool) int {
	// Guard against empty string which would cause infinite loop in whole-word mode
	if search == "" {
		return 0
	}

	count := 0
	lineToCheck := line
	searchTerm := search

	if caseInsensitive {
		lineToCheck = strings.ToLower(line)
		searchTerm = strings.ToLower(search)
	}

	if !wholeWord {
		count = strings.Count(lineToCheck, searchTerm)
		return count
	}

	startIdx := 0
	for {
		idx := strings.Index(lineToCheck[startIdx:], searchTerm)
		if idx == -1 {
			break
		}

		actualIdx := startIdx + idx
		beforeOk := actualIdx == 0 || !isWordChar(rune(lineToCheck[actualIdx-1]))
		afterIdx := actualIdx + len(searchTerm)
		afterOk := afterIdx >= len(lineToCheck) || !isWordChar(rune(lineToCheck[afterIdx]))

		if beforeOk && afterOk {
			count++
		}

		startIdx = actualIdx + 1
	}

	return count
}

func containsWholeWord(text, word string) bool {
	// Guard against empty string which would cause infinite loop
	if word == "" {
		return false
	}

	if !strings.Contains(text, word) {
		return false
	}

	startIdx := 0
	for {
		idx := strings.Index(text[startIdx:], word)
		if idx == -1 {
			return false
		}

		actualIdx := startIdx + idx

		beforeOk := actualIdx == 0 || !isWordChar(rune(text[actualIdx-1]))
		afterIdx := actualIdx + len(word)
		afterOk := afterIdx >= len(text) || !isWordChar(rune(text[afterIdx]))

		if beforeOk && afterOk {
			return true
		}

		startIdx = actualIdx + 1
	}
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func isMultiline(search, replace string) bool {
	return strings.Contains(search, "\n") || strings.Contains(replace, "\n")
}

func countChangedLines(original, modified string) int {
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	changed := 0
	i := 0
	for i < len(origLines) && i < len(modLines) {
		if origLines[i] != modLines[i] {
			changed++
		}
		i++
	}
	changed += len(origLines) - i
	changed += len(modLines) - i

	return changed
}

// replaceContentMultiline performs search/replace on whole-file content, handling all four
// modes (standard, case-insensitive, whole-word, combined) with exclude support.
// Returns the modified content, replacement count, and number of original lines affected.
func replaceContentMultiline(content, search, replace string, caseInsensitive, wholeWord bool, exclude []string) (string, int, int) {
	if search == "" {
		return content, 0, 0
	}

	searchTerm := search
	contentToSearch := content
	if caseInsensitive {
		searchTerm = strings.ToLower(search)
		contentToSearch = strings.ToLower(content)
	}

	var result strings.Builder
	result.Grow(len(content))
	replacements := 0
	affectedLines := make(map[int]bool)
	pos := 0

	for {
		idx := strings.Index(contentToSearch[pos:], searchTerm)
		if idx == -1 {
			result.WriteString(content[pos:])
			break
		}

		matchStart := pos + idx
		matchEnd := matchStart + len(search)

		// Check whole-word boundaries
		if wholeWord {
			beforeOk := matchStart == 0 || !isWordChar(rune(content[matchStart-1]))
			afterOk := matchEnd >= len(content) || !isWordChar(rune(content[matchEnd]))
			if !beforeOk || !afterOk {
				result.WriteString(content[pos : matchStart+1])
				pos = matchStart + 1
				continue
			}
		}

		// Check exclude patterns on the full lines spanning the match
		if len(exclude) > 0 {
			excluded := false
			lineStart := matchStart
			for lineStart > 0 && content[lineStart-1] != '\n' {
				lineStart--
			}
			lineEnd := matchEnd
			for lineEnd < len(content) && content[lineEnd] != '\n' {
				lineEnd++
			}
			spanningText := content[lineStart:lineEnd]

			for _, excl := range exclude {
				exclToCheck := excl
				textToCheck := spanningText
				if caseInsensitive {
					exclToCheck = strings.ToLower(excl)
					textToCheck = strings.ToLower(spanningText)
				}
				if strings.Contains(textToCheck, exclToCheck) {
					excluded = true
					break
				}
			}

			if excluded {
				result.WriteString(content[pos:matchEnd])
				pos = matchEnd
				continue
			}
		}

		// Track affected lines in original content
		startLine := strings.Count(content[:matchStart], "\n")
		matchNewlines := strings.Count(content[matchStart:matchEnd], "\n")
		for l := startLine; l <= startLine+matchNewlines; l++ {
			affectedLines[l] = true
		}

		// Perform replacement
		result.WriteString(content[pos:matchStart])
		result.WriteString(replace)
		pos = matchEnd
		replacements++
	}

	return result.String(), replacements, len(affectedLines)
}

// replaceInFileMultiline handles replacement when search or replace contains newlines.
// Reads the entire file, performs whole-content replacement, and writes back atomically.
func replaceInFileMultiline(path string, config Config) (int, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}

	content := string(data)

	// Detect line ending style
	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}

	// Normalize search/replace to match file's line endings
	search := config.Search
	replace := config.Replace
	if lineEnding == "\r\n" {
		// Normalize any existing \r\n to \n first, then convert all \n to \r\n
		search = strings.ReplaceAll(strings.ReplaceAll(search, "\r\n", "\n"), "\n", "\r\n")
		replace = strings.ReplaceAll(strings.ReplaceAll(replace, "\r\n", "\n"), "\n", "\r\n")
	}

	modified, replacements, linesChanged := replaceContentMultiline(
		content, search, replace,
		config.CaseInsensitive, config.WholeWord, config.Exclude,
	)

	if replacements == 0 {
		return 0, 0, nil
	}

	if !config.DryRun {
		err := writeFileAtomicBytes(path, []byte(modified))
		if err != nil {
			return 0, 0, fmt.Errorf("failed to write file: %w", err)
		}
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Modified: %s (%d replacements in %d lines)\n", path, replacements, linesChanged)
		}
	}

	return linesChanged, replacements, nil
}

// writeFileAtomicBytes writes raw bytes to a file atomically using temp file + rename pattern.
func writeFileAtomicBytes(path string, data []byte) error {
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			resolvedPath = path
		} else {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	mode := os.FileMode(0644)
	if info, err := os.Stat(resolvedPath); err == nil {
		mode = info.Mode()
		if mode&0200 == 0 {
			return fmt.Errorf("file is read-only: %s", resolvedPath)
		}
	}

	dir := filepath.Dir(resolvedPath)
	tmpFile, err := os.CreateTemp(dir, ".repfor-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, resolvedPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// writeFileAtomic writes lines to a file atomically using temp file + rename pattern.
// This prevents data loss if the write fails partway through.
func writeFileAtomic(path string, lines []string, lineEnding string) error {
	// Resolve symlinks so we write to the target, not replace the symlink
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If file doesn't exist (new file), use original path
		if os.IsNotExist(err) {
			resolvedPath = path
		} else {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Get file info to preserve permissions (use default 0644 if file doesn't exist)
	mode := os.FileMode(0644)
	if info, err := os.Stat(resolvedPath); err == nil {
		mode = info.Mode()
		// Check if file is writable (owner write bit)
		if mode&0200 == 0 {
			return fmt.Errorf("file is read-only: %s", resolvedPath)
		}
	}

	// Create temp file in same directory (required for atomic rename)
	dir := filepath.Dir(resolvedPath)
	tmpFile, err := os.CreateTemp(dir, ".repfor-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	writer := bufio.NewWriter(tmpFile)
	for i, line := range lines {
		if i > 0 {
			if _, err := writer.WriteString(lineEnding); err != nil {
				tmpFile.Close()
				return err
			}
		}
		if _, err := writer.WriteString(line); err != nil {
			tmpFile.Close()
			return err
		}
	}

	if len(lines) > 0 {
		if _, err := writer.WriteString(lineEnding); err != nil {
			tmpFile.Close()
			return err
		}
	}

	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		return err
	}

	// Sync to disk before close to ensure data is written
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Preserve original file permissions
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename (on POSIX systems)
	if err := os.Rename(tmpPath, resolvedPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}
