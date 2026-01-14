# Quick Test Guide

## ⚠️ Important: Test Suite Can Hang

The full test suite includes **stress tests and long-running tests** that can take considerable time or hang. Always use the recommended commands below.

## Recommended Test Commands

### Quick Validation (Recommended)
```bash
# Run only fast, essential tests
go test -short -run "^Test(ReplaceInLine|ContainsWholeWord|IsWordChar|CaseInsensitive|WholeWord|Count)" -v
```

### Safe Full Test Run
```bash
# Run all tests except the extremely long-running ones
go test -short -v
```

### Specific Test Categories

**Basic functionality:**
```bash
go test -run "^TestReplaceInLine$" -v
go test -run "^TestReplaceInFile" -v
```

**Unicode edge cases:**
```bash
go test -short -run "Unicode" -v
```

**File operations:**
```bash
go test -short -run "TestReplaceInFile_" -v
```

**Error handling:**
```bash
go test -short -run "TestReplaceInFile_.*Error|NonExistent|ReadOnly" -v
```

### Benchmarks (Safe)
```bash
# Run benchmarks (these don't hang)
go test -bench=BenchmarkReplaceInLine -benchtime=100ms
go test -bench=. -benchtime=100ms -short
```

### Coverage Report
```bash
go test -short -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Tests That May Hang/Take Long

These tests are **skipped** with `-short` flag:

- `TestReplaceInFile_StressTest` - 100,000 lines
- `TestReplaceInFile_MemoryStress` - 500,000 lines
- `TestReplaceInDirectory_StressManyFiles` - 1,000 files
- `TestReplaceInFile_LongRunning` - 1,000,000 lines (2min timeout)
- `TestReplaceInFile_LargeFile` - 100,000 lines
- `TestReplaceInFile_ManyLines` - 1,000,000 lines
- `TestReplaceInLine_RandomInputs` - 1,000 iterations
- `TestContainsWholeWord_RandomInputs` - 1,000 iterations
- Various concurrency tests with 100+ goroutines

## Fuzzing (Advanced)

```bash
# Run fuzzing for 30 seconds
go test -fuzz=FuzzReplaceInLine -fuzztime=30s

# Stop fuzzing with Ctrl+C
```

## Test File Summary

| File | Tests | Safe to Run | Notes |
|------|-------|-------------|-------|
| `main_test.go` | 16 | ✅ Yes | Core functionality, fast |
| `main_advanced_test.go` | 25+ | ✅ Yes with `-short` | Unicode, boundaries |
| `main_concurrency_test.go` | 15+ | ⚠️ Use `-short` | Can be slow |
| `main_fuzz_test.go` | 12+ | ⚠️ Use `-short` | Random tests |
| `main_bench_test.go` | 35 | ✅ Yes | Benchmarks |
| `main_failure_test.go` | 25+ | ✅ Yes with `-short` | Error cases |

## Quick Health Check

Run this to verify the codebase is healthy:

```bash
go test -short -run "^Test(ReplaceInLine|ReplaceInFile|ContainsWholeWord)" -v
```

Should complete in **< 1 second** and show all PASS.

## CI/CD Usage

For continuous integration, use:

```bash
go test -short -race -coverprofile=coverage.out -covermode=atomic
```

This runs:
- All fast tests (stress tests skipped)
- Race detection enabled
- Coverage tracking

## Troubleshooting

**If tests hang:**
1. Press `Ctrl+C` to stop
2. Use `-short` flag to skip long-running tests
3. Run specific test categories instead of all tests

**If you need to run stress tests:**
```bash
# Run ONE stress test at a time
go test -run "^TestReplaceInFile_StressTest$" -v -timeout=5m
```

Always use `-timeout` flag when running stress tests!

## Performance Expectations

| Test Type | Expected Time |
|-----------|---------------|
| Basic tests | < 1 second |
| Short mode (all) | < 5 seconds |
| Individual stress test | 1-30 seconds |
| Full suite (no -short) | ⚠️ 5-15 minutes or may hang |
| Fuzzing | Runs until stopped |

---

**TL;DR:** Always use `go test -short -v` for safety!
