package main

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"
)

// Fuzzing and Property-Based Tests

// FuzzReplaceInLine tests replaceInLine with random inputs
func FuzzReplaceInLine(f *testing.F) {
	// Seed corpus
	f.Add("hello world", "world", "test")
	f.Add("test test test", "test", "exam")
	f.Add("", "foo", "bar")
	f.Add("unicode 世界", "世界", "world")
	f.Add("emoji 👋", "👋", "🌍")

	f.Fuzz(func(t *testing.T, line, search, replace string) {
		// Property: Function should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("replaceInLine panicked: %v", r)
			}
		}()

		result := replaceInLine(line, search, replace, false, false)

		// Property: Result should be valid UTF-8 if inputs are valid UTF-8
		if utf8.ValidString(line) && utf8.ValidString(search) && utf8.ValidString(replace) {
			if !utf8.ValidString(result) {
				t.Errorf("Result is invalid UTF-8 when inputs were valid")
			}
		}

		// Property: If search is empty, result should equal line
		if search == "" {
			if result != line {
				t.Errorf("Empty search should not modify line")
			}
		}

		// Property: If search not in line, result should equal line
		if !strings.Contains(line, search) {
			if result != line {
				t.Errorf("Non-existent search should not modify line")
			}
		}

		// Property: Result should not contain search pattern (unless replace contains it)
		if search != "" && !strings.Contains(replace, search) {
			if strings.Contains(result, search) {
				t.Errorf("Search pattern still exists in result")
			}
		}
	})
}

// FuzzContainsWholeWord tests whole word matching with random inputs
func FuzzContainsWholeWord(f *testing.F) {
	// Seed corpus
	f.Add("hello world", "world")
	f.Add("helloworld", "hello")
	f.Add("test", "test")
	f.Add("", "word")

	f.Fuzz(func(t *testing.T, text, word string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("containsWholeWord panicked: %v", r)
			}
		}()

		result := containsWholeWord(text, word)

		// Property: If word is empty, should return false
		if word == "" {
			if result {
				t.Errorf("Empty word should return false")
			}
		}

		// Property: If word not in text, should return false
		if !strings.Contains(text, word) {
			if result {
				t.Errorf("Word not in text should return false")
			}
		}

		// Property: If text equals word, should return true
		if text == word && word != "" {
			if !result {
				t.Errorf("Exact match should return true")
			}
		}
	})
}

// FuzzCaseInsensitiveReplace tests case-insensitive replacement
func FuzzCaseInsensitiveReplace(f *testing.F) {
	f.Add("Hello World", "hello", "hi")
	f.Add("TEST test Test", "test", "exam")
	f.Add("", "foo", "bar")

	f.Fuzz(func(t *testing.T, line, search, replace string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("caseInsensitiveReplace panicked: %v", r)
			}
		}()

		result := caseInsensitiveReplace(line, search, replace)

		// Property: Result should be valid UTF-8 if inputs are
		if utf8.ValidString(line) && utf8.ValidString(search) && utf8.ValidString(replace) {
			if !utf8.ValidString(result) {
				t.Errorf("Result is invalid UTF-8")
			}
		}

		// Property: Lowercase search should not exist in lowercase result
		if search != "" {
			lowerResult := strings.ToLower(result)
			lowerSearch := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(replace), lowerSearch) {
				if strings.Contains(lowerResult, lowerSearch) {
					t.Errorf("Search pattern still exists (case-insensitive)")
				}
			}
		}
	})
}

// Property-Based Tests (Manual Implementation)

func TestReplaceInLine_Properties(t *testing.T) {
	// Property 1: Idempotency - replacing something that doesn't exist should not change the line
	tests := []struct {
		line    string
		search  string
		replace string
	}{
		{"hello world", "xyz", "abc"},
		{"test", "testing", "exam"},
		{"", "anything", "something"},
		{"unicode 世界", "🌍", "world"},
	}

	for _, tt := range tests {
		result := replaceInLine(tt.line, tt.search, tt.replace, false, false)
		if result != tt.line {
			t.Errorf("Non-existent search modified line: %q -> %q", tt.line, result)
		}
	}
}

func TestReplaceInLine_Commutativity(t *testing.T) {
	// Property: Replacing A with B, then C with D should equal replacing C with D, then A with B
	// (if A and C don't overlap)
	line := "hello world test"

	// First order
	result1 := replaceInLine(line, "hello", "hi", false, false)
	result1 = replaceInLine(result1, "world", "earth", false, false)

	// Second order
	result2 := replaceInLine(line, "world", "earth", false, false)
	result2 = replaceInLine(result2, "hello", "hi", false, false)

	if result1 != result2 {
		t.Errorf("Non-commutative replacement: %q vs %q", result1, result2)
	}
}

func TestReplaceInLine_Associativity(t *testing.T) {
	// Property: Multiple replacements should be associative if patterns don't overlap
	line := "a b c d e"

	// (a->A, b->B), c->C
	temp := replaceInLine(line, "a", "A", false, false)
	temp = replaceInLine(temp, "b", "B", false, false)
	result1 := replaceInLine(temp, "c", "C", false, false)

	// a->A, (b->B, c->C)
	temp = replaceInLine(line, "b", "B", false, false)
	temp = replaceInLine(temp, "c", "C", false, false)
	result2 := replaceInLine(temp, "a", "A", false, false)

	if result1 != result2 {
		t.Errorf("Non-associative replacement: %q vs %q", result1, result2)
	}
}

// Randomized Testing

func TestReplaceInLine_RandomInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping randomized test in short mode")
	}

	rand.Seed(42) // Deterministic randomness

	for i := 0; i < 1000; i++ {
		line := randomString(rand.Intn(1000))
		search := randomString(rand.Intn(50))
		replace := randomString(rand.Intn(50))

		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on random input %d: %v\nline=%q search=%q replace=%q",
						i, r, line, search, replace)
				}
			}()

			result := replaceInLine(line, search, replace, false, false)

			// Basic sanity checks
			if search == "" && result != line {
				t.Errorf("Empty search modified line")
			}

			if !utf8.ValidString(result) && utf8.ValidString(line) {
				t.Errorf("Produced invalid UTF-8 from valid input")
			}
		}()
	}
}

func TestContainsWholeWord_RandomInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping randomized test in short mode")
	}

	rand.Seed(42)

	for i := 0; i < 1000; i++ {
		text := randomString(rand.Intn(500))
		word := randomString(rand.Intn(50))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on random input %d: %v\ntext=%q word=%q",
						i, r, text, word)
				}
			}()

			result := containsWholeWord(text, word)

			// If word not in text, must be false
			if !strings.Contains(text, word) && result {
				t.Errorf("False positive on random input %d", i)
			}

			// If text equals word (and not empty), must be true
			if text == word && word != "" && !result {
				t.Errorf("False negative on exact match %d", i)
			}
		}()
	}
}

func TestWholeWordReplace_RandomInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping randomized test in short mode")
	}

	rand.Seed(42)

	for i := 0; i < 500; i++ {
		line := randomString(rand.Intn(500))
		search := randomString(rand.Intn(30))
		replace := randomString(rand.Intn(30))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on random input %d: %v", i, r)
				}
			}()

			result := wholeWordReplace(line, search, replace)

			// Should produce valid output
			if utf8.ValidString(line) && utf8.ValidString(search) && utf8.ValidString(replace) {
				if !utf8.ValidString(result) {
					t.Errorf("Invalid UTF-8 output on random input %d", i)
				}
			}
		}()
	}
}

func TestCountReplacements_RandomInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping randomized test in short mode")
	}

	rand.Seed(42)

	for i := 0; i < 1000; i++ {
		line := randomString(rand.Intn(500))
		search := randomString(rand.Intn(30))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on random input %d: %v", i, r)
				}
			}()

			count := countReplacements(line, search, false, false)

			// Count should not be negative
			if count < 0 {
				t.Errorf("Negative count on random input %d: %d", i, count)
			}

			// If search is empty or not in line, count should be appropriate
			if search == "" {
				actualCount := strings.Count(line, search)
				if count != actualCount {
					t.Logf("Empty search count difference (expected per strings.Count): %d vs %d",
						count, actualCount)
				}
			} else if !strings.Contains(line, search) {
				if count != 0 {
					t.Errorf("Non-zero count when search not in line: %d", count)
				}
			}
		}()
	}
}

// Edge Case Fuzzing

func TestReplaceInLine_EdgeCaseFuzz(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case fuzz in short mode")
	}

	// Generate edge case inputs
	edgeCases := []string{
		"",
		" ",
		"\n",
		"\t",
		"\r\n",
		"\x00",
		"\xFF\xFE",
		strings.Repeat("a", 10000),
		strings.Repeat("世界", 5000),
		strings.Repeat("👋", 1000),
		"test\x00test",
		"test\ntest\rtest",
	}

	for i, line := range edgeCases {
		for j, search := range edgeCases {
			for k, replace := range edgeCases {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("Panic on edge case %d,%d,%d: %v", i, j, k, r)
						}
					}()

					result := replaceInLine(line, search, replace, false, false)

					// Should complete without panic
					_ = result
				}()
			}
		}
	}
}

// Helper Functions

func randomString(length int) string {
	if length == 0 {
		return ""
	}

	// Character set including ASCII, Unicode, special chars
	charSets := []string{
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"0123456789",
		" \t\n\r",
		"!@#$%^&*()_+-=[]{}|;':\",./<>?",
		"世界你好こんにちは",
		"👋🌍😀🎉",
	}

	var sb strings.Builder
	for i := 0; i < length; i++ {
		charSet := charSets[rand.Intn(len(charSets))]
		if len(charSet) > 0 {
			sb.WriteByte(charSet[rand.Intn(len(charSet))])
		}
	}

	return sb.String()
}

// Invariant Testing

func TestReplaceInLine_Invariants(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		search    string
		replace   string
		invariant func(string, string, string, string) bool
		desc      string
	}{
		{
			"length invariant on no match",
			"hello world",
			"xyz",
			"abc",
			func(line, search, replace, result string) bool {
				return len(result) == len(line)
			},
			"length should not change when no match",
		},
		{
			"empty search invariant",
			"test",
			"",
			"anything",
			func(line, search, replace, result string) bool {
				return result == line
			},
			"empty search should not modify line",
		},
		{
			"UTF-8 validity invariant",
			"hello 世界",
			"世界",
			"world",
			func(line, search, replace, result string) bool {
				return utf8.ValidString(result)
			},
			"result should be valid UTF-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceInLine(tt.line, tt.search, tt.replace, false, false)
			if !tt.invariant(tt.line, tt.search, tt.replace, result) {
				t.Errorf("Invariant violated: %s\nline=%q search=%q replace=%q result=%q",
					tt.desc, tt.line, tt.search, tt.replace, result)
			}
		})
	}
}

// Metamorphic Testing

func TestReplaceInLine_Metamorphic(t *testing.T) {
	// Metamorphic relation: If we replace A with B, then replace B back with A,
	// we should get the original (if A doesn't occur in B)

	tests := []struct {
		line    string
		search  string
		replace string
	}{
		{"hello world", "hello", "goodbye"},
		{"test test test", "test", "exam"},
		{"a b c", "b", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			// Skip if replace contains search
			if strings.Contains(tt.replace, tt.search) {
				t.Skip("Replace contains search, metamorphic property doesn't apply")
			}

			// Forward replacement
			result := replaceInLine(tt.line, tt.search, tt.replace, false, false)

			// Backward replacement
			final := replaceInLine(result, tt.replace, tt.search, false, false)

			if final != tt.line {
				t.Errorf("Metamorphic property violated:\noriginal: %q\nforward:  %q\nbackward: %q",
					tt.line, result, final)
			}
		})
	}
}
