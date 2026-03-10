package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create parent directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

func TestLoadRootConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, "config_repo: config-repo\n")

	cfg, err := LoadRootConfig(dir)
	if err != nil {
		t.Fatalf("LoadRootConfig returned error: %v", err)
	}

	if cfg.ConfigRepo != "config-repo" {
		t.Fatalf("ConfigRepo = %q, want %q", cfg.ConfigRepo, "config-repo")
	}
	if cfg.Git.BranchPrefix != "gitrepoforge/" {
		t.Fatalf("Git.BranchPrefix = %q, want default %q", cfg.Git.BranchPrefix, "gitrepoforge/")
	}
}

func TestLoadCentralConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config/license.yaml", `type: string
required: true
default: mit
enum:
  - mit
  - apache-2.0
`)
	writeFile(t, dir, "config/enabled.yaml", `type: boolean
default: true
`)
	writeFile(t, dir, "outputs/LICENSE.gitrepoforge", `templates:
  - condition: license == "mit"
    template: licenses/mit.tmpl
  - condition: license == "apache-2.0"
    template: licenses/apache-2.0.tmpl
`)
	writeFile(t, dir, "templates/licenses/mit.tmpl", "MIT License\n")
	writeFile(t, dir, "templates/licenses/apache-2.0.tmpl", "Apache License 2.0\n")

	cfg, err := LoadCentralConfig(dir)
	if err != nil {
		t.Fatalf("LoadCentralConfig returned error: %v", err)
	}

	if len(cfg.Definitions) != 2 {
		t.Fatalf("Definitions length = %d, want 2", len(cfg.Definitions))
	}
	if cfg.Definitions[0].Name != "enabled" {
		t.Fatalf("Definitions[0].Name = %q, want %q", cfg.Definitions[0].Name, "enabled")
	}
	if cfg.Definitions[0].Default != true {
		t.Fatalf("Definitions[0].Default = %v, want true", cfg.Definitions[0].Default)
	}
	if cfg.Definitions[1].Name != "license" {
		t.Fatalf("Definitions[1].Name = %q, want %q", cfg.Definitions[1].Name, "license")
	}
	if cfg.Definitions[1].Default != "mit" {
		t.Fatalf("Definitions[1].Default = %v, want %q", cfg.Definitions[1].Default, "mit")
	}
	if len(cfg.Files) != 1 {
		t.Fatalf("Files length = %d, want 1", len(cfg.Files))
	}
	rule := cfg.Files[0]
	if rule.Path != "LICENSE" {
		t.Fatalf("rule.Path = %q, want %q", rule.Path, "LICENSE")
	}
	if len(rule.Templates) != 2 {
		t.Fatalf("Templates length = %d, want 2", len(rule.Templates))
	}
	if !strings.HasSuffix(rule.Templates[0].ResolvedPath, filepath.Join("templates", "licenses", "mit.tmpl")) {
		t.Fatalf("unexpected resolved template path %q", rule.Templates[0].ResolvedPath)
	}
	if rule.Templates[0].Evaluate {
		t.Fatalf("Evaluate = true, want false by default")
	}
}

func TestLoadCentralConfigSupportsAbsentTemplateCandidate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "outputs/justfile.gitrepoforge", `templates:
  - condition: justfile
    template: justfile.tmpl
    evaluate: true
  - absent: true
`)
	writeFile(t, dir, "templates/justfile.tmpl", "test")

	cfg, err := LoadCentralConfig(dir)
	if err != nil {
		t.Fatalf("LoadCentralConfig returned error: %v", err)
	}

	rule := cfg.Files[0]
	if len(rule.Templates) != 2 {
		t.Fatalf("Templates length = %d, want 2", len(rule.Templates))
	}
	if !rule.Templates[0].Evaluate {
		t.Fatalf("Evaluate = false, want true")
	}
	if !rule.Templates[1].Absent {
		t.Fatalf("Absent = false, want true")
	}
}

func TestLoadCentralConfigRejectsReservedConfigName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config/name.yaml", "type: string\n")

	_, err := LoadCentralConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `"name" is reserved`) {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	repoCfg := &RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
	}
	centralCfg := &CentralConfig{
		Definitions: []ConfigDefinition{
			{Name: "license", Type: "string", Default: "mit", HasDefault: true},
			{Name: "enabled", Type: "boolean", Default: true, HasDefault: true},
		},
	}

	ApplyConfigDefaults(repoCfg, centralCfg)

	if repoCfg.Config["license"] != "mit" {
		t.Fatalf("Config[license] = %v, want %q", repoCfg.Config["license"], "mit")
	}
	if repoCfg.Config["enabled"] != true {
		t.Fatalf("Config[enabled] = %v, want true", repoCfg.Config["enabled"])
	}
}

func TestLoadCentralConfigRejectsTemplateOutsideTemplatesDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "outputs/LICENSE.gitrepoforge", `templates:
  - template: ../outside.tmpl
`)

	_, err := LoadCentralConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must stay within templates") {
		t.Fatalf("error %q does not mention template directory boundary", err)
	}
}

func TestLoadCentralConfigRejectsAbsentTemplateWithTemplatePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "outputs/justfile.gitrepoforge", `templates:
  - absent: true
    template: justfile.tmpl
`)

	_, err := LoadCentralConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot also set template") {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestLoadRepoConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RepoConfigFileName, `name: example-repo
default_branch: main
config:
  license: mit
  enabled: true
`)

	cfg, err := LoadRepoConfig(dir)
	if err != nil {
		t.Fatalf("LoadRepoConfig returned error: %v", err)
	}

	if cfg.Name != "example-repo" {
		t.Fatalf("Name = %q, want %q", cfg.Name, "example-repo")
	}
	if cfg.DefaultBranch != "main" {
		t.Fatalf("DefaultBranch = %q, want %q", cfg.DefaultBranch, "main")
	}
	if cfg.Config["license"] != "mit" {
		t.Fatalf("Config[license] = %v, want %q", cfg.Config["license"], "mit")
	}
	if cfg.Config["enabled"] != true {
		t.Fatalf("Config[enabled] = %v, want true", cfg.Config["enabled"])
	}
}
