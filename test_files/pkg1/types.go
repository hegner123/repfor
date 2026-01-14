package pkg1

import (
	"encoding/json"
)

type Match struct {
	Line          int      `json:"line"`
	Content       string   `json:"content"`
	ContextBefore []string `json:"context_before,omitempty"`
	ContextAfter  []string `json:"context_after,omitempty"`
}

type FileMatch struct {
	Path    string  `json:"path"`
	Matches []Match `json:"matches"`
}

type DirectoryResult struct {
	Dir             string      `json:"dir"`
	MatchesFound    int         `json:"matches_found"`
	OriginalMatches int         `json:"original_matches,omitempty"`
	FilteredMatches int         `json:"filtered_matches,omitempty"`
	Files           []FileMatch `json:"files"`
}

type Result struct {
	Directories []DirectoryResult `json:"directories"`
}

type Config struct {
	Dirs            []string
	Search          string
	Ext             string
	Exclude         []string
	CaseInsensitive bool
	WholeWord       bool
	Context         int
	HideFilterStats bool
	CLIMode         bool
}
