package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CentralConfig represents the desired-state configuration repository.
// It defines inputs (repo attributes), file rules, and templates.
type CentralConfig struct {
	Inputs []InputDef `yaml:"inputs"`
	Files  []FileRule `yaml:"files"`
}

// InputDef defines a valid input for per-repo configs.
type InputDef struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Required    bool     `yaml:"required"`
	Enum        []string `yaml:"enum"`
	Default     string   `yaml:"default"`
	Description string   `yaml:"description"`
}

// FileRule defines how a file is managed.
type FileRule struct {
	Path      string      `yaml:"path"`
	Action    string      `yaml:"action"`
	Condition string      `yaml:"condition"`
	Template  string      `yaml:"template"`
	Content   string      `yaml:"content"`
	Blocks    []BlockRule `yaml:"blocks"`
}

// BlockRule defines a managed block within a partially managed file.
type BlockRule struct {
	BeginMarker string `yaml:"begin_marker"`
	EndMarker   string `yaml:"end_marker"`
	Template    string `yaml:"template"`
	Content     string `yaml:"content"`
}

const CentralConfigFileName = "gitrepoforge.yaml"

func LoadCentralConfig(configRepoPath string) (*CentralConfig, error) {
	path := filepath.Join(configRepoPath, CentralConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read central config %s: %w", path, err)
	}
	var cfg CentralConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse central config %s: %w", path, err)
	}
	return &cfg, nil
}
