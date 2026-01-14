package pkg2

import "strings"

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

func getContextBefore(lines []string, currentIdx, count int) []string {
	start := currentIdx - count
	if start < 0 {
		start = 0
	}
	return lines[start:currentIdx]
}

func getContextAfter(lines []string, currentIdx, count int) []string {
	end := currentIdx + count + 1
	if end > len(lines) {
		end = len(lines)
	}
	return lines[currentIdx+1 : end]
}
