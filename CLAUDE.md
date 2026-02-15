# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`repfor` is an MCP server tool for safe, controlled string replacements across files in multiple directories. It's optimized for AI-driven refactoring workflows with exact string matching (no regex) and compact JSON output. The tool operates primarily as an MCP server with an optional CLI mode for testing.

## Build and Development Commands

### Build
```bash
go build -o repfor
```

### Install to System Path
```bash
sudo cp repfor /usr/local/bin/
```

### Run Tests
```bash
go test -v
```

### MCP Mode (Default)
```bash
repfor
```

The MCP server runs by default and communicates via JSON-RPC 2.0 over stdin/stdout. Configuration is in `.mcp.json`.

### CLI Usage (Testing/Scripting)
```bash
repfor --cli --search <string> --replace <string> [options]
```

Required flags:
- `--search` - String to search for
- `--replace` - String to replace with

Optional flags:
- `--cli` - Run in CLI mode (default is MCP server mode)
- `--dir` - Comma-separated list of directories to search (defaults to current directory)
- `--file` - Comma-separated list of files to process (takes precedence over `--dir`)
- `--ext` - File extension filter (e.g., `.go`, `.txt`)
- `--exclude` - Comma-separated list of strings to exclude from replacement
- `--case-insensitive` - Case-insensitive search
- `--whole-word` - Match whole words only
- `--dry-run` - Preview changes without modifying files
- `--recursive` - Recursively search subdirectories
- `--verbose` - Show progress on stderr

## Architecture

### Mode Design

The application operates in two modes determined at startup:

1. **MCP Server Mode (Default)**: JSON-RPC 2.0 server for integration with MCP-compatible clients (Claude Code)
2. **CLI Mode**: Standard command-line tool that outputs JSON results to stdout (requires `--cli` flag)

Mode selection happens in `main()` based on the `--cli` flag. Default behavior is MCP server mode.

### Replacement Algorithm

The replacement flow:
1. `replaceInDirectories()` - Iterates over all provided directories
2. `replaceInDirectory()` - Processes single directory, returns DirectoryResult
3. `replaceInFile()` - Processes individual files, applies replacements, returns statistics

**Key features:**
- **Single-depth by default**: Scans immediate directory contents; use `--recursive` to recurse into subdirectories
- **Multi-directory**: Processes multiple directories in one invocation
- **File mode**: Target specific files by path (takes precedence over directory scanning)
- **Multi-line support**: When search or replace contains `\n`, switches to whole-file processing
- **Exclude filtering**: Skips replacement in lines containing any of the exclude patterns
- **Extension filtering**: Applied before file reading for efficiency
- **In-place modification**: Files are modified directly (no backups)
- **Dry-run mode**: Preview changes without modifying files
- **Exact string matching**: No regex, only literal string replacements

Key implementation details:
- **Single-line path**: Files are read as line slices via `bufio.Scanner`. Search/replace operates per-line.
- **Multi-line path**: When search or replace contains `\n`, `replaceInFileMultiline()` reads the entire file, normalizes line endings in the search/replace strings (converts `\n` to `\r\n` for CRLF files), and performs whole-content replacement via `replaceContentMultiline()`.
- Whole-word matching uses custom `containsWholeWord()` that checks word boundaries using `isWordChar()` (alphanumeric + underscore)
- Case-insensitive search converts both search term and content to lowercase for matching
- Exclude patterns also respect case-insensitive flag
- Four replacement modes (used in both single-line and multi-line paths):
  - Standard: `strings.ReplaceAll()` for simple cases
  - Case-insensitive: `caseInsensitiveReplace()` preserves original case in non-replaced parts
  - Whole-word: `wholeWordReplace()` checks word boundaries
  - Combined: `caseInsensitiveWholeWordReplace()` for both features
- Replacement counting uses `countReplacements()` per line, or affected-line tracking via a line-number set in multi-line mode
- Files are written atomically using `writeFileAtomic()` (line-based) or `writeFileAtomicBytes()` (multi-line) with temp file + rename

### Output Format

**Compact JSON with per-directory structure:**
```json
{"directories":[{"dir":"./pkg","files_modified":2,"lines_changed":5,"total_replacements":8,"files":[{"path":"user.go","lines_changed":3,"replacements":5}]}],"dry_run":false}
```

**Token efficiency:**
- Compact JSON (no newlines): Minimal token usage
- Relative file paths: Reduced path data
- Summary statistics: No need to include full line content
- Per-directory organization: Better AI reasoning

### MCP Protocol Implementation

The MCP server implements three JSON-RPC methods:
- `initialize`: Returns protocol version "2024-11-05" and server capabilities
- `tools/list`: Exposes the "repfor" tool with its schema
- `tools/call`: Executes replacements and returns formatted results

### Data Structures

Key types:
- `Config`: Unified configuration for both CLI and MCP modes
  - `Dirs []string` - List of directories to process
  - `Files []string` - List of specific files to process (takes precedence over Dirs)
  - `Search string` - String to search for
  - `Replace string` - String to replace with
  - `Ext string` - File extension filter
  - `Exclude []string` - List of exclude patterns
  - `CaseInsensitive bool` - Case-insensitive search
  - `WholeWord bool` - Whole-word matching
  - `DryRun bool` - Whether to preview only
  - `Recursive bool` - Recurse into subdirectories
  - `CLIMode bool` - Whether to run in CLI mode
  - `Verbose bool` - Show progress on stderr
  - `ReplaceSet bool` - Tracks if --replace was explicitly provided (allows empty string)
- `Result`: Top-level result with `Directories []DirectoryResult` and `DryRun bool`
- `DirectoryResult`: Per-directory results with:
  - `Dir string` - Directory path
  - `FilesModified int` - Number of files modified
  - `LinesChanged int` - Total lines changed
  - `TotalReplacements int` - Total replacements made
  - `Files []FileModification` - File modification details
- `FileModification`: Contains relative file path, lines changed, and replacement count

## Safety Considerations

### No Regex Policy

This tool deliberately avoids regex for safety and predictability:
- Regex can cause accidental matches and unintended replacements
- Exact string matching is easier to reason about
- Aligns with user's CLAUDE.md constraint against regex for editing tasks
- Makes dry-run previews more reliable

### File Safety

- Files are only modified if replacement is successful
- Atomic writes via temp file + rename prevent partial writes
- Dry-run mode allows previewing before applying
- Extension filtering limits scope
- Exclude patterns prevent unwanted replacements
- Single-depth scanning by default limits blast radius (recursive is opt-in)

### Recommended Workflow

1. Use checkfor to search first
2. Plan exclude patterns based on results
3. Run repfor with --dry-run
4. Review dry-run output
5. Run repfor without --dry-run
6. Verify with checkfor again

## Comparison with checkfor

repfor is built on checkfor's architecture but focused on replacements:

**Similarities:**
- Same multi-directory scanning (single-depth by default, recursive optional)
- Same filtering options (ext, case-insensitive, whole-word, exclude)
- Same MCP server + CLI mode design
- Same compact JSON output philosophy

**Differences:**
- checkfor outputs match locations with content and context
- repfor outputs modification statistics without content
- checkfor is read-only
- repfor modifies files in-place
- repfor adds dry-run mode
- repfor requires both search and replace parameters

## Important Notes

- The tool is **single-depth by default** - use `--recursive` or the `recursive` MCP parameter to recurse into subdirectories
- Default mode is **MCP server** - runs without flags
- CLI mode requires `--cli` flag
- Both `--search` and `--replace` are required
- File paths in results are **relative to each directory**
- All JSON output is **compact** (no indentation/newlines) for token efficiency
- Replacements are **in-place** with no backups
- Multi-line search/replace activates when search or replace contains `\n`
- Multi-directory support enables controlled replacements across specific packages
- Warnings for unreadable/unwritable files go to stderr
- MCP mode expects one JSON-RPC request per line on stdin
- Use dry-run mode to preview changes before applying

## Testing Strategy

When adding tests, focus on:
- Core replacement functions (case-insensitive, whole-word, combined)
- Multi-line replacement (basic, dry-run, exclude, case-insensitive, whole-word, CRLF)
- File modification (dry-run vs actual writes)
- Filter statistics tracking
- MCP JSON-RPC protocol compliance
- Integration tests (multi-directory, exclude patterns, extension filtering)
- Edge cases (empty files, files with no matches, permission errors)
