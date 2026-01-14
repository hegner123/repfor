# Test Files

This directory contains test files for testing repfor functionality.

## Structure

- `pkg1/` - Contains type definitions and handler code
- `pkg2/` - Contains search and utility functions

## Common Patterns to Test

### Function Names
- `searchDirectories` -> Could replace with `processDirectories`
- `searchDirectory` -> Could replace with `processDirectory`
- `searchFile` -> Could replace with `processFile`

### Type Names
- `Match` -> Could replace with `SearchMatch`
- `Config` -> Could replace with `Configuration`
- `Result` -> Could replace with `SearchResult`

### Variable Names
- `config` -> Could replace with `cfg`
- `result` -> Could replace with `res`
- `err` -> Keep as is (Go convention)

### String Literals
- `"error"` -> Could replace with `"failure"`
- `"Parse error"` -> Could replace with `"Parsing failed"`

## Test Scenarios

1. **Simple replacement**: Replace "search" with "process"
2. **Whole-word only**: Replace "word" (not "password", not "keywords")
3. **Case-insensitive**: Replace "error" matching "Error", "ERROR", etc.
4. **Extension filter**: Only replace in .go files (not .md)
5. **Exclude filter**: Replace "result" but exclude lines with "dirResult"
