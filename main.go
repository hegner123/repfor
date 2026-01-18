package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Search          string
	Replace         string
	Ext             string
	Exclude         []string
	CaseInsensitive bool
	WholeWord       bool
	DryRun          bool
	CLIMode         bool
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
	flag.StringVar(&config.Search, "search", "", "String to search for (required)")
	flag.StringVar(&config.Replace, "replace", "", "String to replace with (required)")
	flag.StringVar(&config.Ext, "ext", "", "File extension to filter (e.g., .go, .txt)")
	flag.StringVar(&excludeStr, "exclude", "", "Comma-separated list of strings to exclude from replacement")
	flag.BoolVar(&config.CaseInsensitive, "case-insensitive", false, "Perform case-insensitive search")
	flag.BoolVar(&config.WholeWord, "whole-word", false, "Match whole words only")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Preview changes without modifying files")

	flag.Parse()

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

func runCLI(config Config) {
	if config.Search == "" {
		fmt.Fprintln(os.Stderr, "Error: --search is required")
		flag.Usage()
		os.Exit(1)
	}

	if config.Replace == "" {
		fmt.Fprintln(os.Stderr, "Error: --replace is required")
		flag.Usage()
		os.Exit(1)
	}

	result, err := replaceInDirectories(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func runMCPServer() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
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
				Description: "Search and replace strings in files across directories. Single-depth (non-recursive) scanning with in-place file modifications. Supports extension filtering, case-insensitive search, whole-word matching, and exclude filters.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"dir": {
							Type:        "array",
							Description: "Array of directory paths to search. Can also accept a single string for backwards compatibility. Defaults to current directory if not provided.",
						},
						"search": {
							Type:        "string",
							Description: "String pattern to search for",
						},
						"replace": {
							Type:        "string",
							Description: "String to replace matches with",
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

	for _, dir := range config.Dirs {
		dirResult, err := replaceInDirectory(dir, config)
		if err != nil {
			return nil, err
		}

		result.Directories = append(result.Directories, *dirResult)
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

func replaceInFile(path string, config Config) (int, int, error) {
	// Early exit: if search equals replace, it's a no-op
	if config.Search == config.Replace {
		return 0, 0, nil
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

	var lines []string
	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle very long lines (default is 64KB, set to 10MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
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
		err := writeFile(path, modifiedLines)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to write file: %w", err)
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
	searchLower := strings.ToLower(search)
	result := ""
	remaining := line

	for {
		lineLower := strings.ToLower(remaining)
		idx := strings.Index(lineLower, searchLower)
		if idx == -1 {
			result += remaining
			break
		}

		result += remaining[:idx]
		result += replace
		remaining = remaining[idx+len(search):]
	}

	return result
}

func wholeWordReplace(line, search, replace string) string {
	result := ""
	remaining := line
	searchLen := len(search)

	for {
		idx := strings.Index(remaining, search)
		if idx == -1 {
			result += remaining
			break
		}

		beforeOk := idx == 0 || !isWordChar(rune(remaining[idx-1]))
		afterIdx := idx + searchLen
		afterOk := afterIdx >= len(remaining) || !isWordChar(rune(remaining[afterIdx]))

		if beforeOk && afterOk {
			result += remaining[:idx]
			result += replace
			remaining = remaining[afterIdx:]
		} else {
			result += remaining[:idx+1]
			remaining = remaining[idx+1:]
		}
	}

	return result
}

func caseInsensitiveWholeWordReplace(line, search, replace string) string {
	result := ""
	remaining := line
	searchLower := strings.ToLower(search)
	searchLen := len(search)

	for {
		lineLower := strings.ToLower(remaining)
		idx := strings.Index(lineLower, searchLower)
		if idx == -1 {
			result += remaining
			break
		}

		beforeOk := idx == 0 || !isWordChar(rune(remaining[idx-1]))
		afterIdx := idx + searchLen
		afterOk := afterIdx >= len(remaining) || !isWordChar(rune(remaining[afterIdx]))

		if beforeOk && afterOk {
			result += remaining[:idx]
			result += replace
			remaining = remaining[afterIdx:]
		} else {
			result += remaining[:idx+1]
			remaining = remaining[idx+1:]
		}
	}

	return result
}

func countReplacements(line, search string, caseInsensitive, wholeWord bool) int {
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

func writeFile(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", path, cerr)
		}
	}()

	writer := bufio.NewWriter(file)
	for i, line := range lines {
		if i > 0 {
			if _, err := writer.WriteString("\n"); err != nil {
				return err
			}
		}
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	if len(lines) > 0 {
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}
