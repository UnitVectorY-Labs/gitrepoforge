package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	RepoConfigFileName       = ".gitrepoforge"
	ManagedFilesManifestName = ".gitrepoforge-managed-files.yaml"
)

const (
	// PullRequestNo disables pull request creation.
	PullRequestNo = "NO"
	// PullRequestGitHubCLI creates pull requests using the GitHub CLI.
	PullRequestGitHubCLI = "GITHUB_CLI"
)

var gitPlaceholderRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// RepoConfig represents the per-repo config dotfile (.gitrepoforge).
type RepoConfig struct {
	Name          string                 `yaml:"name"`
	DefaultBranch string                 `yaml:"default_branch"`
	Config        map[string]interface{} `yaml:"config"`
}

func LoadRepoConfig(repoPath string) (*RepoConfig, error) {
	path := filepath.Join(repoPath, RepoConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo config %s: %w", path, err)
	}
	var cfg RepoConfig
	if err := unmarshalYAMLKnownFields(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse repo config %s: %w", path, err)
	}
	return &cfg, nil
}

// RepoConfigExists returns true if a .gitrepoforge file exists in the repo.
func RepoConfigExists(repoPath string) bool {
	path := filepath.Join(repoPath, RepoConfigFileName)
	_, err := os.Stat(path)
	return err == nil
}

func (rc *RepoConfig) PlaceholderValues() map[string]string {
	values := map[string]string{
		"name":           rc.Name,
		"default_branch": rc.DefaultBranch,
	}
	for key, value := range rc.Config {
		values[key] = fmt.Sprintf("%v", value)
	}
	return values
}

func ExtractGitPlaceholders(value string) []string {
	matches := gitPlaceholderRegex.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	var placeholders []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		placeholders = append(placeholders, name)
	}
	return placeholders
}

func substituteGitPlaceholders(value string, values map[string]string) string {
	result := value
	for key, replacement := range values {
		result = strings.ReplaceAll(result, "{{"+key+"}}", replacement)
	}
	return result
}
