package targets

import (
	"path/filepath"
	"sort"
	"strings"
)

// RepoNamesFromChangedFiles extracts unique repository names from a list of
// changed file paths within the config repo. It strips the directory prefix
// and file extension, returning the bare repository names sorted alphabetically.
func RepoNamesFromChangedFiles(changedFiles []string) []string {
	seen := make(map[string]bool)
	var names []string

	for _, f := range changedFiles {
		if f == "" {
			continue
		}
		base := filepath.Base(f)
		name := strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}
