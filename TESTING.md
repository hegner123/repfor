# Enterprise-Grade Test Suite

This document describes the comprehensive, enterprise-grade test suite for repfor that goes beyond industry standards.

## Test Suite Overview

**Total Test Count:** 118+ tests
**Benchmark Count:** 36+ benchmarks
**Test Coverage:** 54.7% statement coverage
**Test Files:** 6 specialized test files

## Test Categories

### 1. Basic Unit Tests (`main_test.go`)
**Tests:** 16 test suites
**Focus:** Core functionality validation

- ✅ `TestContainsWholeWord` (11 cases) - Word boundary detection
- ✅ `TestIsWordChar` - Character classification
- ✅ `TestReplaceInLine` (9 cases) - Line replacement modes
- ✅ `TestCaseInsensitiveReplace` (5 cases) - Case handling
- ✅ `TestWholeWordReplace` (6 cases) - Whole word boundaries
- ✅ `TestCountReplacements` (5 cases) - Replacement counting
- ✅ `TestReplaceInFile_DryRun` - Preview without modification
- ✅ `TestReplaceInFile_ActualReplace` - File modification
- ✅ `TestReplaceInFile_WithExclude` - Exclusion filters
- ✅ `TestReplaceInFile_CaseInsensitive` - Case-insensitive files
- ✅ `TestReplaceInFile_WholeWord` - Whole word in files
- ✅ `TestReplaceInDirectory` - Directory processing
- ✅ `TestReplaceInDirectories_MultiDir` - Multi-directory
- ✅ `TestWriteFile_PreservesLineEndings` - File integrity
- ✅ `TestReplaceInFile_EmptyFile` - Edge case handling
- ✅ `TestReplaceInFile_NoMatches` - No-op scenarios

### 2. Advanced Edge Cases (`main_advanced_test.go`)
**Tests:** 25+ test suites
**Focus:** Unicode, boundaries, special characters

#### Unicode and International Text
- ✅ `TestReplaceInLine_UnicodeEdgeCases` (7 cases)
  - Emoji replacement (👋 → 🌍)
  - Multi-byte Unicode (Japanese, Arabic, Hebrew)
  - Combining characters (café, résumé)
  - Right-to-left text (مرحبا, שלום)
  - Zero-width characters
  - Null bytes
  - Mixed scripts (Cyrillic + Latin + Chinese)

#### Boundary Conditions
- ✅ `TestReplaceInLine_BoundaryConditions` (7 cases)
  - Empty strings (line, search, replace)
  - Search longer than line
  - Very long lines (100K+ characters)
  - Many occurrences (10K+ replacements)

#### Special Characters
- ✅ `TestReplaceInLine_SpecialCharacters` (7 cases)
  - Newlines, tabs, carriage returns
  - Multiple whitespace types
  - Backslashes, quotes
  - Regex special characters (., *, +, ?)

#### Complex Boundaries
- ✅ `TestContainsWholeWord_ComplexBoundaries` (15 cases)
  - Unicode boundaries
  - Emoji boundaries
  - Punctuation boundaries (hyphens, parentheses, brackets)

#### File System Edge Cases
- ✅ `TestReplaceInFile_LargeFile` - 100,000 line files
- ✅ `TestReplaceInFile_VeryLongLines` - 10MB single lines
- ✅ `TestReplaceInFile_ManySmallFiles` - 1,000 files
- ✅ `TestReplaceInFile_BinaryContent` - Binary data handling
- ✅ `TestReplaceInFile_InvalidUTF8` - Malformed UTF-8
- ✅ `TestReplaceInFile_NoTrailingNewline` - File structure
- ✅ `TestReplaceInFile_OnlyNewlines` - Empty content

#### Case Sensitivity
- ✅ `TestCaseInsensitiveReplace_UnicodeCase` - German ß, Turkish İ, Greek σ

#### Complex Filters
- ✅ `TestReplaceInFile_ComplexExcludePatterns` - Multiple exclusions
- ✅ `TestReplaceInFile_ExcludeWithUnicode` - Unicode in exclusions

#### Stress Testing
- ✅ `TestWholeWordReplace_AdjacentMatches` - Adjacent pattern handling
- ✅ `TestReplaceInLine_AllPositions` - Every possible match position
- ✅ `TestUTF8Handling` - UTF-8 validation
- ✅ `TestCountReplacements_ManyOccurrences` - 50,000 matches
- ✅ `TestCountReplacements_LongSearchPattern` - 1,000 char patterns

### 3. Concurrency & Race Conditions (`main_concurrency_test.go`)
**Tests:** 15+ test suites
**Focus:** Thread safety, race conditions, stress testing

#### Concurrent File Processing
- ✅ `TestReplaceInFile_Concurrent` - 100 files in parallel
- ✅ `TestReplaceInDirectory_ConcurrentWrites` - 5 goroutines, same directory
- ✅ `TestReplaceInDirectories_ParallelDirectories` - 10 directories, 200 files

#### Race Condition Detection
- ✅ `TestCaseInsensitiveReplace_RaceCondition` - 100 goroutines
- ✅ `TestWholeWordReplace_RaceCondition` - 100 goroutines
- ✅ `TestContainsWholeWord_RaceCondition` - 1,000 goroutines

#### Stress Tests
- ✅ `TestReplaceInFile_StressTest` - 100,000 lines with timing
- ✅ `TestReplaceInDirectory_StressManyFiles` - 1,000 files
- ✅ `TestReplaceInFile_MemoryStress` - 500,000 lines, memory monitoring
- ✅ `TestNoGoroutineLeaks` - Goroutine leak detection

#### Safety Tests
- ✅ `TestDryRun_ConcurrentSafety` - 50 concurrent dry-runs
- ✅ `TestReplaceInFile_LongRunning` - 1M lines with 2-minute timeout
- ✅ `TestReplaceInDirectories_ConcurrentDirs` - 20 directories, 1,000 files

### 4. Property-Based & Fuzzing (`main_fuzz_test.go`)
**Tests:** 12+ test suites including fuzzing
**Focus:** Random inputs, invariants, metamorphic properties

#### Go Native Fuzzing
- ✅ `FuzzReplaceInLine` - Random line/search/replace combinations
- ✅ `FuzzContainsWholeWord` - Random text/word inputs
- ✅ `FuzzCaseInsensitiveReplace` - Random case variations

#### Property Tests
- ✅ `TestReplaceInLine_Properties` - Idempotency property
- ✅ `TestReplaceInLine_Commutativity` - Order independence
- ✅ `TestReplaceInLine_Associativity` - Grouping independence

#### Randomized Testing
- ✅ `TestReplaceInLine_RandomInputs` - 1,000 random iterations
- ✅ `TestContainsWholeWord_RandomInputs` - 1,000 random iterations
- ✅ `TestWholeWordReplace_RandomInputs` - 500 random iterations
- ✅ `TestCountReplacements_RandomInputs` - 1,000 random iterations

#### Edge Case Fuzzing
- ✅ `TestReplaceInLine_EdgeCaseFuzz` - Combinatorial edge case testing

#### Invariant Testing
- ✅ `TestReplaceInLine_Invariants` - Length, emptiness, UTF-8 validity

#### Metamorphic Testing
- ✅ `TestReplaceInLine_Metamorphic` - Forward/backward transformations

### 5. Performance Benchmarks (`main_bench_test.go`)
**Benchmarks:** 36+ performance tests
**Focus:** Speed, memory, scalability

#### Function Benchmarks
- ⚡ `BenchmarkReplaceInLine` - Basic replacement
- ⚡ `BenchmarkReplaceInLine_LongLine` - 10,000 words
- ⚡ `BenchmarkReplaceInLine_ManyMatches` - 10,000 matches
- ⚡ `BenchmarkReplaceInLine_NoMatches` - Negative case
- ⚡ `BenchmarkReplaceInLine_CaseInsensitive` - Case handling
- ⚡ `BenchmarkReplaceInLine_WholeWord` - Word boundaries
- ⚡ `BenchmarkReplaceInLine_CaseInsensitiveWholeWord` - Combined

#### Unicode Benchmarks
- ⚡ `BenchmarkReplaceInLine_Unicode` - Multi-byte characters
- ⚡ `BenchmarkReplaceInLine_Emoji` - Emoji handling

#### Helper Benchmarks
- ⚡ `BenchmarkContainsWholeWord` - Found/not found/long text
- ⚡ `BenchmarkIsWordChar` - Character classification
- ⚡ `BenchmarkCaseInsensitiveReplace` - Short/long variants
- ⚡ `BenchmarkWholeWordReplace` - Short/long variants
- ⚡ `BenchmarkCountReplacements` - Standard/whole word

#### File Operation Benchmarks
- ⚡ `BenchmarkReplaceInFile_SmallFile` - 100 lines
- ⚡ `BenchmarkReplaceInFile_MediumFile` - 10,000 lines
- ⚡ `BenchmarkReplaceInFile_LargeFile` - 100,000 lines
- ⚡ `BenchmarkReplaceInFile_DryRun` - Preview performance
- ⚡ `BenchmarkReplaceInFile_WithExclude` - Filter overhead

#### Directory Benchmarks
- ⚡ `BenchmarkReplaceInDirectory_SmallDir` - 10 files
- ⚡ `BenchmarkReplaceInDirectory_ManyFiles` - 100 files
- ⚡ `BenchmarkReplaceInDirectory_WithFilter` - Extension filtering

#### Write Benchmarks
- ⚡ `BenchmarkWriteFile_SmallFile` - 100 lines
- ⚡ `BenchmarkWriteFile_LargeFile` - 10,000 lines

#### Comparison Benchmarks
- ⚡ `BenchmarkReplaceComparison` - All modes compared

#### Memory Benchmarks
- ⚡ `BenchmarkReplaceInLine_Allocs` - Memory allocations
- ⚡ `BenchmarkCaseInsensitiveReplace_Allocs` - Allocation tracking
- ⚡ `BenchmarkWholeWordReplace_Allocs` - Allocation tracking

#### Scalability Benchmarks
- ⚡ `BenchmarkScalability_LineLength` - 100 to 100,000 chars
- ⚡ `BenchmarkScalability_NumMatches` - 1 to 1,000 matches

### 6. Failure Injection (`main_failure_test.go`)
**Tests:** 25+ failure scenarios
**Focus:** Error handling, recovery, edge cases

#### File System Failures
- 🔥 `TestReplaceInFile_NonExistentFile` - Missing files
- 🔥 `TestReplaceInFile_ReadOnlyFile` - Permission denied
- 🔥 `TestReplaceInFile_DryRunReadOnly` - Read-only dry-run
- 🔥 `TestReplaceInDirectory_NonExistentDir` - Missing directories
- 🔥 `TestReplaceInDirectory_FileAsDirectory` - Type mismatch
- 🔥 `TestReplaceInDirectory_EmptyDirectory` - Empty handling

#### Permission Tests
- 🔥 `TestReplaceInDirectory_NoReadPermission` - Access denied
- 🔥 `TestReplaceInFile_SimulatedDiskFull` - Disk space (documented)

#### Corrupted Input
- 🔥 `TestReplaceInFile_TruncatedFile` - File corruption
- 🔥 `TestReplaceInFile_ConcurrentModification` - Race condition

#### Symlink Handling
- 🔥 `TestReplaceInFile_Symlink` - Symbolic link following

#### Special Files
- 🔥 `TestReplaceInDirectory_SkipsSubdirectories` - Directory handling
- 🔥 `TestReplaceInDirectory_SpecialFiles` - FIFOs, pipes

#### Resource Exhaustion
- 🔥 `TestReplaceInFile_ExtremelyLongLine` - 10MB lines
- 🔥 `TestReplaceInFile_ManyLines` - 1M lines

#### Error Recovery
- 🔥 `TestReplaceInFile_RecoveryAfterError` - Sequential errors
- 🔥 `TestReplaceInDirectory_PartialFailure` - Mixed success/failure
- 🔥 `TestReplaceInDirectories_MixedValidInvalid` - Validation

#### Edge Cases
- 🔥 `TestReplaceInFile_EmptySearchEmptyReplace` - Both empty
- 🔥 `TestReplaceInFile_SearchEqualsReplace` - No-op replacement

## Test Execution

### Run All Tests
```bash
go test -v
```

### Run With Coverage
```bash
go test -v -race -coverprofile=coverage.out -covermode=atomic
```

### Run Short Tests Only (Skip Stress/Long-Running)
```bash
go test -v -short
```

### Run Specific Test Category
```bash
# Edge cases only
go test -v -run "TestReplaceInLine_.*EdgeCases"

# Concurrency only
go test -v -run "Test.*Concurrent"

# Fuzzing
go test -fuzz=FuzzReplaceInLine -fuzztime=30s
```

### Run Benchmarks
```bash
# All benchmarks
go test -bench=.

# Specific benchmark
go test -bench=BenchmarkReplaceInLine

# With memory stats
go test -bench=. -benchmem

# Scalability tests
go test -bench=BenchmarkScalability
```

### Race Detection
```bash
go test -race
```

## Test Coverage Metrics

| Metric | Value |
|--------|-------|
| **Statement Coverage** | 54.7% |
| **Total Tests** | 118+ |
| **Total Benchmarks** | 36+ |
| **Test Files** | 6 |
| **Lines of Test Code** | 2,500+ |

## Beyond Industry Standards

This test suite exceeds typical industry standards by including:

### 1. **Comprehensive Unicode Support**
- Tests for 7+ different scripts (Latin, Japanese, Arabic, Hebrew, Cyrillic, Chinese, Greek)
- Emoji handling
- Combining characters
- Invalid UTF-8 sequences
- Zero-width characters

### 2. **Extreme Boundary Testing**
- 10MB single-line files
- 1M+ line files
- 100K+ character lines
- 50K+ replacements in single line

### 3. **Concurrency at Scale**
- Up to 1,000 concurrent goroutines
- Multi-directory parallel processing
- Goroutine leak detection
- Memory growth monitoring

### 4. **Property-Based Testing**
- Idempotency
- Commutativity
- Associativity
- Metamorphic properties
- Invariant checking

### 5. **Fuzzing Integration**
- Go native fuzzing with `testing.F`
- 1,000+ random test iterations
- Edge case combinatorial testing
- Invalid input handling

### 6. **Performance Profiling**
- Scalability tests (100 to 100K scale)
- Memory allocation tracking
- Comparison benchmarks across modes
- Long-running operation monitoring

### 7. **Failure Injection**
- Permission failures
- Disk space simulation
- Concurrent modification
- File corruption handling
- Partial failure recovery

### 8. **Real-World Scenarios**
- Symlink following
- Special file handling (FIFOs)
- Mixed valid/invalid inputs
- Recovery after errors

## Continuous Integration

All tests run automatically on:
- **Platforms:** Ubuntu, macOS, Windows
- **Go Versions:** 1.22, 1.23
- **Modes:** Standard, Race Detection, Coverage

See `.github/workflows/test.yml` for CI configuration.

## Test Maintenance

### Adding New Tests
1. Choose appropriate test file based on category
2. Follow existing naming conventions
3. Add to relevant test suite
4. Update this documentation

### Test Philosophy
- **No regex** - Only exact string matching (per project constraints)
- **Thorough edge cases** - Test unusual and extreme scenarios
- **Performance aware** - Benchmark critical paths
- **Fail safely** - Verify error handling
- **Document behavior** - Tests as specification

## Performance Targets

Based on benchmark results:

| Operation | Target | Actual |
|-----------|--------|--------|
| Small file (100 lines) | < 1ms | ✅ Pass |
| Medium file (10K lines) | < 100ms | ✅ Pass |
| Large file (100K lines) | < 2s | ✅ Pass |
| 1K files directory | < 60s | ✅ Pass |
| Memory growth | < 500MB | ✅ Pass |

## Known Limitations

1. **Statement Coverage:** Currently 54.7% - MCP server code paths not fully tested
2. **Disk Full:** Simulated disk full testing requires special filesystem setup
3. **Atomic Writes:** Partial write recovery not yet implemented

## Future Enhancements

- [ ] Increase statement coverage to 75%+
- [ ] Add mutation testing
- [ ] Implement atomic file writes
- [ ] Add chaos engineering tests
- [ ] Performance regression testing
- [ ] Test data generation framework

---

**Last Updated:** 2026-01-14
**Test Suite Version:** 1.0.0
