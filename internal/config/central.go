package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CentralConfig represents the desired-state configuration repository.
// Inputs are loaded from individual files under inputs/.
// File rules are loaded from individual files under outputs/.
type CentralConfig struct {
	Inputs []InputDef
	Files  []FileRule
}

// InputDef defines a valid input for per-repo configs.
// Each input is stored as its own file: inputs/<name>.yaml
type InputDef struct {
	Name        string   `yaml:"-"`
	Type        string   `yaml:"type"`
	Required    bool     `yaml:"required"`
	Enum        []string `yaml:"enum"`
	Default     string   `yaml:"default"`
	Description string   `yaml:"description"`
}

// FileRule defines how an output file is managed.
// Each rule is stored as its own file: outputs/<path>.gitrepoforge
type FileRule struct {
	Path      string      `yaml:"-"`
	Mode      string      `yaml:"mode"`
	Condition string      `yaml:"condition"`
	Template  string      `yaml:"template"`
	Blocks    []BlockRule `yaml:"blocks"`
}

// BlockRule defines a managed block within a partially managed file.
type BlockRule struct {
	BeginMarker string `yaml:"begin_marker"`
	EndMarker   string `yaml:"end_marker"`
	Template    string `yaml:"template"`
}

const (
	InputsDir       = "inputs"
	OutputsDir      = "outputs"
	OutputFileSuffix = ".gitrepoforge"
)

// LoadCentralConfig loads the central config from the config repo by scanning
// the inputs/ and outputs/ directories for individual definition files.
func LoadCentralConfig(configRepoPath string) (*CentralConfig, error) {
	cfg := &CentralConfig{}

	inputs, err := loadInputDefs(filepath.Join(configRepoPath, InputsDir))
	if err != nil {
		return nil, fmt.Errorf("failed to load inputs: %w", err)
	}
	cfg.Inputs = inputs

	files, err := loadOutputRules(filepath.Join(configRepoPath, OutputsDir))
	if err != nil {
		return nil, fmt.Errorf("failed to load outputs: %w", err)
	}
	cfg.Files = files

	return cfg, nil
}

// loadInputDefs scans the inputs/ directory for YAML files.
// Each file defines one input; the filename (without .yaml) is the input name.
func loadInputDefs(inputsDir string) ([]InputDef, error) {
	if _, err := os.Stat(inputsDir); os.IsNotExist(err) {
		return nil, nil // no inputs directory is valid (empty inputs)
	}

	entries, err := os.ReadDir(inputsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read inputs directory %s: %w", inputsDir, err)
	}

	var inputs []InputDef
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		path := filepath.Join(inputsDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file %s: %w", path, err)
		}

		var def InputDef
		if err := yaml.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("failed to parse input file %s: %w", path, err)
		}
		def.Name = name
		inputs = append(inputs, def)
	}

	// Sort by name for deterministic ordering
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Name < inputs[j].Name
	})

	return inputs, nil
}

// loadOutputRules walks the outputs/ directory tree for .gitrepoforge files.
// Each file defines one output rule; the path relative to outputs/ minus the
// .gitrepoforge suffix is the target file path.
func loadOutputRules(outputsDir string) ([]FileRule, error) {
	if _, err := os.Stat(outputsDir); os.IsNotExist(err) {
		return nil, nil // no outputs directory is valid (empty rules)
	}

	var rules []FileRule
	err := filepath.Walk(outputsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), OutputFileSuffix) {
			return nil
		}

		relPath, err := filepath.Rel(outputsDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
		// Strip the .gitrepoforge suffix to get the target file path
		targetPath := strings.TrimSuffix(relPath, OutputFileSuffix)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read output file %s: %w", path, err)
		}

		var rule FileRule
		if err := yaml.Unmarshal(data, &rule); err != nil {
			return fmt.Errorf("failed to parse output file %s: %w", path, err)
		}
		rule.Path = targetPath
		if rule.Mode == "" {
			rule.Mode = "create" // default mode
		}
		rules = append(rules, rule)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk outputs directory %s: %w", outputsDir, err)
	}

	// Sort by path for deterministic ordering
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Path < rules[j].Path
	})

	return rules, nil
}
