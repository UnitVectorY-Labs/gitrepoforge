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
// Config definitions are loaded from individual files under config/.
// File rules are loaded from individual files under outputs/.
type CentralConfig struct {
	RootDir     string
	Definitions []ConfigDefinition
	Files       []FileRule
}

// ConfigDefinition defines a valid config value for per-repo configs.
// Each definition is stored as its own file: config/<name>.yaml
type ConfigDefinition struct {
	Name        string      `yaml:"-"`
	Type        string      `yaml:"type"`
	Required    bool        `yaml:"required"`
	Enum        []string    `yaml:"enum"`
	Default     interface{} `yaml:"default"`
	HasDefault  bool        `yaml:"-"`
	Description string      `yaml:"description"`
}

// FileRule defines how an output file is managed.
// Each rule is stored as its own file: outputs/<path>.gitrepoforge
type FileRule struct {
	Path      string        `yaml:"-"`
	Mode      string        `yaml:"mode"`
	Templates []TemplateRef `yaml:"templates"`
}

// TemplateRef selects a template file from templates/.
type TemplateRef struct {
	Condition    string `yaml:"condition"`
	Template     string `yaml:"template"`
	Evaluate     bool   `yaml:"evaluate"`
	Absent       bool   `yaml:"absent"`
	ResolvedPath string `yaml:"-"`
}

const (
	ConfigDir        = "config"
	OutputsDir       = "outputs"
	TemplatesDir     = "templates"
	OutputFileSuffix = ".gitrepoforge"
)

var reservedConfigNames = map[string]struct{}{
	"name":           {},
	"default_branch": {},
	"config":         {},
}

// LoadCentralConfig loads the central config from the config repo by scanning
// the config/ and outputs/ directories for individual definition files.
func LoadCentralConfig(configRepoPath string) (*CentralConfig, error) {
	cfg := &CentralConfig{
		RootDir: configRepoPath,
	}

	definitions, err := loadConfigDefinitions(filepath.Join(configRepoPath, ConfigDir))
	if err != nil {
		return nil, fmt.Errorf("failed to load config definitions: %w", err)
	}
	cfg.Definitions = definitions

	files, err := loadOutputRules(
		filepath.Join(configRepoPath, OutputsDir),
		filepath.Join(configRepoPath, TemplatesDir),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load outputs: %w", err)
	}
	cfg.Files = files

	return cfg, nil
}

// loadConfigDefinitions scans the config/ directory for YAML files.
// Each file defines one config value; the filename (without .yaml) is the key.
func loadConfigDefinitions(configDir string) ([]ConfigDefinition, error) {
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory %s: %w", configDir, err)
	}

	var definitions []ConfigDefinition
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		path := filepath.Join(configDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		var def ConfigDefinition
		if err := yaml.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
		def.Name = name
		if err := validateDefinition(def); err != nil {
			return nil, fmt.Errorf("invalid config file %s: %w", path, err)
		}
		definitions = append(definitions, def)
	}

	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Name < definitions[j].Name
	})

	return definitions, nil
}

// loadOutputRules walks the outputs/ directory tree for .gitrepoforge files.
// Each file defines one output rule; the path relative to outputs/ minus the
// .gitrepoforge suffix is the target file path.
func loadOutputRules(outputsDir, templatesDir string) ([]FileRule, error) {
	if _, err := os.Stat(outputsDir); os.IsNotExist(err) {
		return nil, nil
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
			return fmt.Errorf("unexpected file %s in %s: output rules must end with %s", path, outputsDir, OutputFileSuffix)
		}

		relPath, err := filepath.Rel(outputsDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
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
			rule.Mode = "create"
		}
		for i := range rule.Templates {
			if err := validateTemplateRef(rule.Templates[i]); err != nil {
				return fmt.Errorf("output file %s: %w", path, err)
			}
			if rule.Templates[i].Absent {
				continue
			}
			resolved, err := resolveTemplatePath(templatesDir, rule.Templates[i].Template)
			if err != nil {
				return fmt.Errorf("output file %s: %w", path, err)
			}
			rule.Templates[i].ResolvedPath = resolved
		}
		rules = append(rules, rule)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk outputs directory %s: %w", outputsDir, err)
	}

	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Path < rules[j].Path
	})

	return rules, nil
}

func resolveTemplatePath(templatesDir, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("template is required")
	}
	if filepath.IsAbs(ref) {
		return "", fmt.Errorf("template %q must be relative to %s", ref, TemplatesDir)
	}

	cleanRef := filepath.Clean(ref)
	if cleanRef == "." || strings.HasPrefix(cleanRef, "..") {
		return "", fmt.Errorf("template %q must stay within %s", ref, TemplatesDir)
	}

	return filepath.Join(templatesDir, cleanRef), nil
}

func validateTemplateRef(ref TemplateRef) error {
	if ref.Absent {
		if ref.Template != "" {
			return fmt.Errorf("absent template candidate cannot also set template")
		}
		if ref.Evaluate {
			return fmt.Errorf("absent template candidate cannot set evaluate")
		}
		return nil
	}

	if ref.Template == "" {
		return fmt.Errorf("template is required unless absent is true")
	}

	return nil
}

func (d *ConfigDefinition) UnmarshalYAML(node *yaml.Node) error {
	type plain ConfigDefinition
	var value plain
	if err := node.Decode(&value); err != nil {
		return err
	}
	*d = ConfigDefinition(value)

	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "default" {
			d.HasDefault = true
			break
		}
	}

	return nil
}

func validateDefinition(def ConfigDefinition) error {
	if _, reserved := reservedConfigNames[def.Name]; reserved {
		return fmt.Errorf("%q is reserved and cannot be used as a config key", def.Name)
	}
	if def.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !def.HasDefault {
		return nil
	}

	if err := validateDefaultValue(def); err != nil {
		return err
	}

	return nil
}

func IsReservedConfigName(name string) bool {
	_, reserved := reservedConfigNames[name]
	return reserved
}

func ApplyConfigDefaults(repoCfg *RepoConfig, centralCfg *CentralConfig) {
	if repoCfg.Config == nil {
		repoCfg.Config = map[string]interface{}{}
	}

	for _, def := range centralCfg.Definitions {
		if !def.HasDefault {
			continue
		}
		if _, exists := repoCfg.Config[def.Name]; exists {
			continue
		}
		repoCfg.Config[def.Name] = def.Default
	}
}

func validateDefaultValue(def ConfigDefinition) error {
	switch def.Type {
	case "string":
		value, ok := def.Default.(string)
		if !ok {
			return fmt.Errorf("default must be a string")
		}
		if len(def.Enum) > 0 {
			for _, allowed := range def.Enum {
				if allowed == value {
					return nil
				}
			}
			return fmt.Errorf("default %q is not one of: %s", value, strings.Join(def.Enum, ", "))
		}
	case "boolean":
		if _, ok := def.Default.(bool); !ok {
			return fmt.Errorf("default must be a boolean")
		}
	case "number":
		switch def.Default.(type) {
		case int, int64, float64:
			return nil
		default:
			return fmt.Errorf("default must be a number")
		}
	case "list":
		if _, ok := def.Default.([]interface{}); !ok {
			return fmt.Errorf("default must be a list")
		}
	default:
		return fmt.Errorf("unsupported config type %q", def.Type)
	}

	return nil
}
