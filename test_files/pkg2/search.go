package pkg2

import (
	"os"
	"path/filepath"
	"strings"
)

func processDirectories(config Config) (*Result, error) {
	result := &Result{
		Directories: make([]DirectoryResult, 0, len(config.Dirs)),
	}

	for _, dir := range config.Dirs {
		dirResult, err := searchDirectory(dir, config)
		if err != nil {
			return nil, err
		}

		result.Directories = append(result.Directories, *dirResult)
	}

	return result, nil
}

func searchDirectory(dir string, config Config) (*DirectoryResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	dirResult := &DirectoryResult{
		Dir:   dir,
		Files: make([]FileMatch, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		if config.Ext != "" && !strings.HasSuffix(filename, config.Ext) {
			continue
		}

		fullPath := filepath.Join(dir, filename)
		matches, originalCount, filteredCount, err := searchFile(fullPath, config)
		if err != nil {
			continue
		}

		if !config.HideFilterStats && len(config.Exclude) > 0 {
			dirResult.OriginalMatches += originalCount
			dirResult.FilteredMatches += filteredCount
		}

		if len(matches) > 0 {
			dirResult.Files = append(dirResult.Files, FileMatch{
				Path:    filename,
				Matches: matches,
			})
			dirResult.MatchesFound += len(matches)
		}
	}

	return dirResult, nil
}

func searchFile(path string, config Config) ([]Match, int, int, error) {
	// Implementation here
	return nil, 0, 0, nil
}
