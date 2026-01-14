# START.md - repfor Project

## Project Overview

`repfor` is an MCP server tool for safe, controlled string replacements across files in multiple directories. It's optimized for AI-driven refactoring workflows with exact string matching (no regex) and compact JSON output.

**Key Features:**
- Non-recursive directory scanning (single depth)
- Multi-directory support
- Exact string matching (no regex)
- Dry-run preview mode
- Extension and exclude filtering
- MCP server + CLI modes

## Onboarding

When starting work on this project, memorize these key paths and structures:

**Core Files:**
- `/Users/home/Documents/Code/Go_dev/repfor/main.go` - Entry point, mode selection, CLI parsing
- `/Users/home/Documents/Code/Go_dev/repfor/main_test.go` - Test suite
- `/Users/home/Documents/Code/Go_dev/repfor/CLAUDE.md` - Project documentation and architecture
- `/Users/home/Documents/Code/Go_dev/repfor/START.md` - This file (onboarding guide)
- `/Users/home/Documents/Code/Go_dev/repfor/.mcp.json` - MCP server configuration
- `/Users/home/Documents/Code/Go_dev/repfor/go.mod` - Go module definition

**Key Directories:**
- `/Users/home/Documents/Code/Go_dev/repfor/` - Project root

**Read these files to understand the project:**
1. `CLAUDE.md` - Complete architecture, design decisions, and comparison with checkfor
2. `.mcp.json` - MCP server configuration example

## Quick Start

### Build
```bash
go build -o repfor
```

### Install to System
```bash
sudo cp repfor /usr/local/bin/
```

### Run Tests
```bash
go test -v
```

### Run as MCP Server (Default)
```bash
repfor
```

### Run in CLI Mode
```bash
repfor --cli --search "oldText" --replace "newText" --dir ./src --ext .go --dry-run
```

## Architecture Overview

### Mode Design
- **MCP Server Mode (Default)**: JSON-RPC 2.0 server for Claude Code integration
- **CLI Mode**: Command-line tool with JSON output (requires `--cli` flag)

### Replacement Flow
1. `replaceInDirectories()` - Iterates over directories
2. `replaceInDirectory()` - Processes single directory
3. `replaceInFile()` - Processes individual files, applies replacements

### Key Implementation Details
- Non-recursive scanning (skips subdirectories)
- In-place file modification (no backups)
- Four replacement modes: standard, case-insensitive, whole-word, combined
- Exclude filtering prevents unwanted replacements
- Compact JSON output for token efficiency

## Data Structures

**Config**: Unified configuration
- `Dirs []string` - Directories to process
- `Search string` - String to search for
- `Replace string` - String to replace with
- `Exclude []string` - Exclude patterns
- `DryRun bool` - Preview mode
- `CLIMode bool` - CLI vs MCP mode

**Result**: Top-level result
- `Directories []DirectoryResult` - Per-directory results
- `DryRun bool` - Whether this was a dry run

**DirectoryResult**: Per-directory statistics
- `Dir string` - Directory path
- `FilesModified int` - Files modified count
- `LinesChanged int` - Total lines changed
- `TotalReplacements int` - Total replacements made
- `Files []FileModification` - File details

## MCP Protocol

Three JSON-RPC methods:
- `initialize` - Returns protocol version and capabilities
- `tools/list` - Exposes the "repfor" tool schema
- `tools/call` - Executes replacements

## Safety Features

1. **No Regex**: Exact string matching only for predictability
2. **Dry-run Mode**: Preview before applying changes
3. **Exclude Patterns**: Skip lines containing specific strings
4. **Extension Filtering**: Limit scope to specific file types
5. **Single-depth Scanning**: Limits blast radius

## Recommended Workflow

1. Use `checkfor` MCP tool to search first
2. Plan exclude patterns based on results
3. Run `repfor` with `--dry-run`
4. Review dry-run output
5. Run `repfor` without `--dry-run`
6. Verify with `checkfor` again

## Comparison with checkfor

Built on checkfor's architecture but focused on replacements:

**Same:**
- Multi-directory, single-depth scanning
- Filtering options (ext, case-insensitive, whole-word, exclude)
- MCP + CLI mode design
- Compact JSON output

**Different:**
- checkfor: read-only, outputs match locations with content
- repfor: modifies files, outputs modification statistics

## Testing Strategy

Focus areas:
- Core replacement functions (all four modes)
- File modification (dry-run vs actual writes)
- Filter statistics tracking
- MCP JSON-RPC protocol compliance
- Integration tests (multi-directory, exclude patterns)
- Edge cases (empty files, no matches, permissions)

## Important Notes

- Default mode is MCP server (no flags needed)
- CLI mode requires `--cli` flag
- Both `--search` and `--replace` are required
- File paths in results are relative to each directory
- All JSON output is compact (no whitespace)
- Replacements are in-place with no backups
- Tool is single-depth only (no recursion)
- Use dry-run to preview changes

## Next Steps

1. Read `CLAUDE.md` for complete architecture details
2. Run `go test -v` to verify setup
3. Build the project with `go build -o repfor`
4. Try a dry-run replacement to see the output format
5. Review the MCP configuration in `.mcp.json`
