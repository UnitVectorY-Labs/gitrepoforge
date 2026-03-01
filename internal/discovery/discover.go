package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverRepos scans the workspace directory for Git repositories.
// It returns sorted repo directory paths. Dot-named directories are included.
// Directories matching exclude patterns are skipped.
func DiscoverRepos(workspaceDir string, excludes []string) ([]string, error) {
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, err
	}

	var repos []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isExcluded(name, excludes) {
			continue
		}
		repoPath := filepath.Join(workspaceDir, name)
		if isGitRepo(repoPath) {
			repos = append(repos, repoPath)
		}
	}
	sort.Strings(repos)
	return repos, nil
}

// isGitRepo checks if a directory is a Git repository by looking for a .git directory.
func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// isExcluded checks if a name matches any of the exclude patterns.
// Supports simple glob matching via filepath.Match.
func isExcluded(name string, excludes []string) bool {
	for _, pattern := range excludes {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		if strings.EqualFold(name, pattern) {
			return true
		}
	}
	return false
}
