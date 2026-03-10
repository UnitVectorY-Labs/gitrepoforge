package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const RepoConfigFileName = ".gitrepoforge"

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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
