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
	ManagedFilesManifestName = ".managedfiles"
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
	Manifest      string                 `yaml:"manifest"`
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
	cfg.Manifest, err = validateAndNormalizeManifestPath(cfg.Manifest)
	if err != nil {
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

func ResolveManifestPath(rootCfg *RootConfig, repoCfg *RepoConfig) string {
	if repoCfg != nil && repoCfg.Manifest != "" {
		return repoCfg.Manifest
	}
	if rootCfg != nil && rootCfg.Manifest != "" {
		return rootCfg.Manifest
	}
	return ManagedFilesManifestName
}

func validateAndNormalizeManifestPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("manifest must be a relative path")
	}
	cleanPath := filepath.Clean(trimmed)
	if cleanPath == "." || cleanPath == "" {
		return "", fmt.Errorf("manifest must not be empty")
	}
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("manifest must stay within the repository")
	}
	return cleanPath, nil
}
