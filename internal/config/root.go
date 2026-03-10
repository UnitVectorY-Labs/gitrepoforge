package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RootConfig represents the root config dotfile (.gitrepoforge-config)
// that lives in the checkout root (workspace directory).
type RootConfig struct {
	ConfigRepo   string   `yaml:"config_repo"`
	Excludes     []string `yaml:"excludes"`
	BranchPrefix string   `yaml:"branch_prefix"`
	CreatePR     bool     `yaml:"create_pr"`
}

const RootConfigFileName = ".gitrepoforge-config"

func LoadRootConfig(workspaceDir string) (*RootConfig, error) {
	path := filepath.Join(workspaceDir, RootConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read root config %s: %w", path, err)
	}
	var cfg RootConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse root config %s: %w", path, err)
	}
	if cfg.ConfigRepo == "" {
		return nil, fmt.Errorf("root config %s: config_repo is required", path)
	}
	if cfg.BranchPrefix == "" {
		cfg.BranchPrefix = "gitrepoforge/"
	}
	return &cfg, nil
}

// ResolveConfigRepoPath resolves the config repo path (relative to workspace or absolute).
func (rc *RootConfig) ResolveConfigRepoPath(workspaceDir string) string {
	if filepath.IsAbs(rc.ConfigRepo) {
		return rc.ConfigRepo
	}
	return filepath.Join(workspaceDir, rc.ConfigRepo)
}
