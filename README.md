# repfor

[![Tests](https://github.com/hegner123/repfor/actions/workflows/test.yml/badge.svg)](https://github.com/hegner123/repfor/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/go-1.23-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/hegner123/repfor)](https://goreportcard.com/report/github.com/hegner123/repfor)

An MCP server tool for searching and replacing strings in files across directories. Designed for safe, token-efficient refactoring workflows with AI assistance.

## Features

- **MCP server by default** - Optimized for Claude Code integration
- **In-place replacements** - Modify files directly with exact string matching
- **Multi-directory support** - Process multiple directories in single-depth or recursive scans
- **File mode** - Target specific files by path instead of scanning directories
- **Multi-line support** - Search and replace patterns spanning multiple lines using `\n`
- **Recursive scanning** - Optionally recurse into subdirectories
- **Dry-run mode** - Preview changes before applying them
- **Compact JSON output** - Token-efficient summary statistics
- **Exclude filtering** - Prevent replacements in lines containing specific patterns
- **File extension filtering** - Target specific file types
- **Case-insensitive search** - Optional case-insensitive matching
- **Whole-word matching** - Avoid false positives from partial matches
- **Safe replacements** - No regex, only exact string matching to prevent accidental changes

## Installation

### 1. Build from source

```bash
go build -o repfor
```

### 2. Install to PATH (Required for MCP mode)

Installing repfor to your system PATH allows MCP server integration with Claude Code.

**System-wide installation** (recommended):
```bash
sudo cp repfor /usr/local/bin/
```

**User-local installation** (if you don't have sudo access):
```bash
mkdir -p ~/bin
cp repfor ~/bin/
# Add ~/bin to PATH if not already (add to ~/.bashrc or ~/.zshrc):
export PATH="$HOME/bin:$PATH"
```

Verify installation:
```bash
repfor --help
```

### 3. MCP Server Setup (For Claude Code Integration)

To use repfor as an MCP tool in Claude Code:

**Step 1:** Add to your Claude Code MCP configuration (project `.mcp.json` or global `~/.claude/mcp.json`):

```json
{
  "mcpServers": {
    "repfor": {
      "command": "repfor"
    }
  }
}
```

**Step 2:** Restart Claude Code or reload MCP servers

**Step 3 (Optional but Recommended):** Optimize Claude Code's tool selection

Add this to your global `~/.claude/CLAUDE.md` to help Claude Code automatically choose repfor for replacement tasks:

```markdown
## Tool Usage - Replace Optimization

### When to use repfor (MCP tool)
Use the `repfor` tool for safe, controlled string replacements across multiple directories. This tool performs single-depth (non-recursive) scanning with exact string matching (no regex).

**Use repfor when:**
- Refactoring code by renaming variables, functions, or types
- Updating string literals or configuration values
- Replacing deprecated function calls with new ones
- Making consistent changes across multiple files
- You need token-efficient output with summary statistics

**Example:**

repfor tool with:
- dir: ["/path/to/pkg/handlers", "/path/to/pkg/models"] (optional, defaults to current directory)
- search: "oldFunctionName" (required)
- replace: "newFunctionName" (required)
- ext: ".go" (optional, filters by extension)
- exclude: ["oldFunctionNames", "testOldFunction"] (optional, prevents replacement in these contexts)
- case_insensitive: false (optional)
- whole_word: true (optional, recommended to avoid partial matches)
- dry_run: false (optional, set true to preview changes)

### When NOT to use repfor
- Complex pattern replacements requiring regex (use manual editing)
- Single file edits (use Edit tool for precision)
- When you need to see full file context before/after
```

## Usage

### MCP Mode (Default)

By default, repfor runs as an MCP server:

```bash
repfor
```

This starts the MCP server and waits for JSON-RPC requests on stdin. This is the primary mode for Claude Code integration.

### CLI Mode

To use repfor in CLI mode (for testing or scripting), add the `--cli` flag:

```bash
repfor --cli --search <string> --replace <string> [options]
```

### Required Flags

- `--search` - String to search for
- `--replace` - String to replace with

### Optional Flags

- `--cli` - Run in CLI mode (default is MCP server mode)
- `--dir` - Comma-separated list of directories to search (defaults to current directory)
- `--file` - Comma-separated list of files to process (takes precedence over `--dir`)
- `--ext` - File extension to filter (e.g., `.go`, `.txt`, `.js`)
- `--exclude` - Comma-separated list of strings to exclude from replacement
- `--case-insensitive` - Perform case-insensitive search
- `--whole-word` - Match whole words only (recommended)
- `--dry-run` - Preview changes without modifying files
- `--recursive` - Recursively search subdirectories
- `--verbose` - Show progress on stderr

## Examples

### Basic replacement (current directory)
```bash
repfor --cli --search "oldFunc" --replace "newFunc"
```

### Replace in specific directory with dry-run
```bash
repfor --cli --dir ./src --search "oldFunc" --replace "newFunc" --dry-run
```

### Replace across multiple directories
```bash
repfor --cli --dir "./pkg/handlers,./pkg/models,./pkg/services" --search "UserModel" --replace "User" --ext .go
```

### Replace with exclude filter
```bash
repfor --cli --dir ./pkg --search "m.Table" --replace "m.TableName" --exclude "m.TableNames,m.TablePrefix" --ext .go
```

### Case-insensitive replacement
```bash
repfor --cli --search "todo" --replace "FIXME" --case-insensitive
```

### Whole word matching (recommended)
```bash
repfor --cli --search "log" --replace "logger" --whole-word --ext .go
```

### Dry-run to preview changes
```bash
repfor --cli --dir ./services --search "deprecated" --replace "updated" --dry-run
```

## Best Practices

- **Always use dry-run first:** Preview changes with `--dry-run` before applying
- **Use whole-word matching:** Add `--whole-word` to avoid partial matches (e.g., "log" matching "logger")
- **Use exclude filters:** Prevent replacements in unwanted contexts
- **Target specific extensions:** Use `--ext` to limit scope and improve safety
- **Process directories incrementally:** Replace in one directory at a time for large refactorings
- **Verify with checkfor:** Use the checkfor tool before and after to verify completeness

## Output Format

The tool outputs compact JSON with per-directory summary statistics:

### Basic replacement
```json
{"directories":[{"dir":"./pkg/handlers","files_modified":2,"lines_changed":5,"total_replacements":8,"files":[{"path":"user.go","lines_changed":3,"replacements":5},{"path":"auth.go","lines_changed":2,"replacements":3}]}]}
```

### Dry-run mode
```json
{"directories":[{"dir":"./src","files_modified":3,"lines_changed":7,"total_replacements":12,"files":[{"path":"main.go","lines_changed":2,"replacements":4}]}],"dry_run":true}
```

### Pretty-printed Example (for documentation)

```json
{
  "directories": [
    {
      "dir": "./pkg/handlers",
      "files_modified": 2,
      "lines_changed": 5,
      "total_replacements": 8,
      "files": [
        {
          "path": "user.go",
          "lines_changed": 3,
          "replacements": 5
        },
        {
          "path": "auth.go",
          "lines_changed": 2,
          "replacements": 3
        }
      ]
    },
    {
      "dir": "./pkg/models",
      "files_modified": 0,
      "lines_changed": 0,
      "total_replacements": 0,
      "files": []
    }
  ],
  "dry_run": false
}
```

### Output Fields

**Top Level:**
- `directories` - Array of directory results
- `dry_run` - Boolean indicating if this was a dry-run (omitted if false)

**Per Directory:**
- `dir` - Directory path
- `files_modified` - Number of files modified in this directory
- `lines_changed` - Total lines changed in this directory
- `total_replacements` - Total number of replacements made in this directory
- `files` - Array of modified files

**Per File:**
- `path` - File path relative to directory
- `lines_changed` - Number of lines changed in this file
- `replacements` - Number of replacements made in this file

## Safety Features

- **No regex:** Only exact string matching to prevent accidental changes
- **Dry-run mode:** Preview changes before applying
- **Exclude filters:** Prevent replacements in specific contexts
- **Whole-word matching:** Avoid partial matches
- **Single-depth by default:** Non-recursive to limit scope (use `--recursive` to opt in)
- **Extension filtering:** Target specific file types
- **Atomic writes:** Temp file + rename pattern prevents data loss on write failures

## Workflow Integration

### Recommended workflow with checkfor

1. **Search first:** Use checkfor to find all instances
   ```bash
   checkfor --cli --search "oldFunc" --ext .go
   ```

2. **Plan exclusions:** Identify contexts where replacement should be avoided

3. **Dry-run:** Preview changes with repfor
   ```bash
   repfor --cli --search "oldFunc" --replace "newFunc" --ext .go --dry-run
   ```

4. **Apply changes:** Run repfor without dry-run
   ```bash
   repfor --cli --search "oldFunc" --replace "newFunc" --ext .go
   ```

5. **Verify:** Use checkfor to confirm no instances remain
   ```bash
   checkfor --cli --search "oldFunc" --ext .go
   ```

## Architecture

- **Default mode:** MCP server (JSON-RPC 2.0 over stdin/stdout)
- **CLI mode:** Direct JSON output (requires `--cli` flag)
- **Single-depth by default:** Optional recursive scanning with `--recursive`
- **Multi-directory:** Controlled replacements across specific directories
- **File mode:** Target specific files by path instead of directory scanning
- **Multi-line:** Search/replace patterns spanning multiple lines via `\n`
- **In-place modification:** Files are modified directly (no backups created)
- **Exact matching:** No regex patterns, only literal string matching

## Exit Codes

- `0` - Success (replacements made or not)
- `1` - Error (invalid arguments, directory not found, file write error, etc.)

## Comparison with checkfor

repfor is the companion tool to checkfor:

| Feature | checkfor | repfor |
|---------|----------|--------|
| Purpose | Search only | Search and replace |
| Output | Match locations with context | Summary statistics |
| File modification | No | Yes (in-place) |
| Dry-run mode | N/A | Yes |
| Use case | Verification | Refactoring |

Use checkfor to verify, use repfor to refactor.

## License

MIT
