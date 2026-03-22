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
	if cfg.Git.PullRequest != PullRequestNo {
		t.Fatalf("Git.PullRequest = %q, want default %q", cfg.Git.PullRequest, PullRequestNo)
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

func TestLoadCentralConfigSupportsObjectDefinitions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config/docs.yaml", `type: object
required: true
description: Documentation settings.
`)
	writeFile(t, dir, "config/docs/enabled.yaml", `type: boolean
default: true
`)
	writeFile(t, dir, "config/docs/domain.yaml", `type: string
required: true
`)

	cfg, err := LoadCentralConfig(dir)
	if err != nil {
		t.Fatalf("LoadCentralConfig returned error: %v", err)
	}

	if len(cfg.Definitions) != 1 {
		t.Fatalf("Definitions length = %d, want 1", len(cfg.Definitions))
	}
	def := cfg.Definitions[0]
	if def.Name != "docs" {
		t.Fatalf("Definition name = %q, want %q", def.Name, "docs")
	}
	if def.Type != "object" {
		t.Fatalf("Definition type = %q, want %q", def.Type, "object")
	}
	if len(def.Attributes) != 2 {
		t.Fatalf("Attributes length = %d, want 2", len(def.Attributes))
	}
	if def.Attributes[0].Name != "domain" {
		t.Fatalf("Attributes[0].Name = %q, want %q", def.Attributes[0].Name, "domain")
	}
	if def.Attributes[1].Name != "enabled" {
		t.Fatalf("Attributes[1].Name = %q, want %q", def.Attributes[1].Name, "enabled")
	}
	if def.Attributes[1].Default != true {
		t.Fatalf("Attributes[1].Default = %v, want true", def.Attributes[1].Default)
	}
}

func TestLoadCentralConfigRejectsObjectDefinitionWithoutAttributeDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config/docs.yaml", "type: object\n")

	_, err := LoadCentralConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "object config definitions require an attribute directory") {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestLoadCentralConfigSupportsUnconditionalTemplateCandidate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "outputs/.github/workflows/add-to-project.yml.gitrepoforge", `templates:
  - template: .github/workflows/add-to-project.yml
`)
	writeFile(t, dir, "templates/.github/workflows/add-to-project.yml", "name: add to project\n")

	cfg, err := LoadCentralConfig(dir)
	if err != nil {
		t.Fatalf("LoadCentralConfig returned error: %v", err)
	}

	if len(cfg.Files) != 1 {
		t.Fatalf("Files length = %d, want 1", len(cfg.Files))
	}
	rule := cfg.Files[0]
	if rule.Path != filepath.Join(".github", "workflows", "add-to-project.yml") {
		t.Fatalf("rule.Path = %q, want workflow path", rule.Path)
	}
	if len(rule.Templates) != 1 {
		t.Fatalf("Templates length = %d, want 1", len(rule.Templates))
	}
	if rule.Templates[0].Condition != "" {
		t.Fatalf("Condition = %q, want empty", rule.Templates[0].Condition)
	}
	if !strings.HasSuffix(rule.Templates[0].ResolvedPath, filepath.Join("templates", ".github", "workflows", "add-to-project.yml")) {
		t.Fatalf("unexpected resolved template path %q", rule.Templates[0].ResolvedPath)
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

func TestApplyConfigDefaultsNestedObject(t *testing.T) {
	repoCfg := &RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"docs": map[string]interface{}{
				"domain": "foo.example.com",
			},
		},
	}
	centralCfg := &CentralConfig{
		Definitions: []ConfigDefinition{
			{
				Name: "docs",
				Type: "object",
				Attributes: []ConfigDefinition{
					{Name: "enabled", Type: "boolean", Default: true, HasDefault: true},
					{Name: "domain", Type: "string", Required: true},
				},
			},
		},
	}

	ApplyConfigDefaults(repoCfg, centralCfg)

	docs, ok := repoCfg.Config["docs"].(map[string]interface{})
	if !ok {
		t.Fatalf("Config[docs] has unexpected type %T", repoCfg.Config["docs"])
	}
	if docs["enabled"] != true {
		t.Fatalf("Config[docs][enabled] = %v, want true", docs["enabled"])
	}
	if docs["domain"] != "foo.example.com" {
		t.Fatalf("Config[docs][domain] = %v, want %q", docs["domain"], "foo.example.com")
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

func TestLoadRootConfigGitSection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
create_branch: true
branch_name: "ops/{{name}}"
commit: true
commit_message: "custom commit"
push: true
remote: upstream
pull_request: GITHUB_CLI
return_to_original_branch: true
delete_branch: true
`)

	cfg, err := LoadRootConfig(dir)
	if err != nil {
		t.Fatalf("LoadRootConfig returned error: %v", err)
	}

	if !cfg.Git.CreateBranch {
		t.Fatalf("Git.CreateBranch = false, want true")
	}
	if cfg.Git.BranchName != "ops/{{name}}" {
		t.Fatalf("Git.BranchName = %q, want %q", cfg.Git.BranchName, "ops/{{name}}")
	}
	if !cfg.Git.Commit {
		t.Fatalf("Git.Commit = false, want true")
	}
	if cfg.Git.CommitMessage != "custom commit" {
		t.Fatalf("Git.CommitMessage = %q, want %q", cfg.Git.CommitMessage, "custom commit")
	}
	if !cfg.Git.Push {
		t.Fatalf("Git.Push = false, want true")
	}
	if cfg.Git.Remote != "upstream" {
		t.Fatalf("Git.Remote = %q, want %q", cfg.Git.Remote, "upstream")
	}
	if cfg.Git.PullRequest != PullRequestGitHubCLI {
		t.Fatalf("Git.PullRequest = %q, want %q", cfg.Git.PullRequest, PullRequestGitHubCLI)
	}
	if !cfg.Git.ReturnToOriginalBranch {
		t.Fatalf("Git.ReturnToOriginalBranch = false, want true")
	}
	if !cfg.Git.DeleteBranch {
		t.Fatalf("Git.DeleteBranch = false, want true")
	}
}

func TestLoadRootConfigGitDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, "config_repo: config-repo\n")

	cfg, err := LoadRootConfig(dir)
	if err != nil {
		t.Fatalf("LoadRootConfig returned error: %v", err)
	}

	if cfg.Git.CreateBranch {
		t.Fatalf("Git.CreateBranch = true, want false")
	}
	if cfg.Git.Commit {
		t.Fatalf("Git.Commit = true, want false")
	}
	if cfg.Git.Push {
		t.Fatalf("Git.Push = true, want false")
	}
	if cfg.Git.PullRequest != PullRequestNo {
		t.Fatalf("Git.PullRequest = %q, want %q", cfg.Git.PullRequest, PullRequestNo)
	}
	if cfg.Git.ReturnToOriginalBranch {
		t.Fatalf("Git.ReturnToOriginalBranch = true, want false")
	}
	if cfg.Git.DeleteBranch {
		t.Fatalf("Git.DeleteBranch = true, want false (default)")
	}
}

func TestLoadRootConfigRejectsLegacyBranchPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
branch_prefix: legacy/
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadRootConfigRejectsLegacyCreatePR(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
create_pr: true
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadRootConfigRejectsUnknownGitField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
bootstrap_commit_message: "legacy"
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadRootConfigRejectsInvalidPullRequest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
pull_request: INVALID
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "git.pull_request") {
		t.Fatalf("error %q does not mention git.pull_request", err)
	}
}

func TestLoadRootConfigRejectsPRWithoutPush(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
push: false
pull_request: GITHUB_CLI
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "requires git.push") {
		t.Fatalf("error %q does not mention push constraint", err)
	}
}

func TestLoadRootConfigRejectsDeleteBranchWithoutReturn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
return_to_original_branch: false
delete_branch: true
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete_branch requires") {
		t.Fatalf("error %q does not mention delete_branch constraint", err)
	}
}

func TestLoadRootConfigRejectsReturnWithoutCreateBranch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
return_to_original_branch: true
`)

	_, err := LoadRootConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create_branch") {
		t.Fatalf("error %q does not mention create_branch constraint", err)
	}
}

func TestLoadRootConfigPushFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
push: false
`)

	cfg, err := LoadRootConfig(dir)
	if err != nil {
		t.Fatalf("LoadRootConfig returned error: %v", err)
	}

	if cfg.Git.Push {
		t.Fatalf("Git.Push = true, want false")
	}
}

func TestLoadRootConfigPullRequestCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RootConfigFileName, `config_repo: config-repo
push: true
remote: origin
pull_request: github_cli
`)

	cfg, err := LoadRootConfig(dir)
	if err != nil {
		t.Fatalf("LoadRootConfig returned error: %v", err)
	}

	if cfg.Git.PullRequest != PullRequestGitHubCLI {
		t.Fatalf("Git.PullRequest = %q, want %q (normalized to upper case)", cfg.Git.PullRequest, PullRequestGitHubCLI)
	}
}

func TestLoadCentralConfigRejectsUnexpectedOutputFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "outputs/.github/workflows/add-to-project.yml.gitrepofroge", `templates:
  - template: .github/workflows/add-to-project.yml
`)

	_, err := LoadCentralConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "must end with .gitrepoforge") {
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

func TestLoadRepoConfigRejectsUnknownTopLevelField(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, RepoConfigFileName, `name: example-repo
default_branch: main
git:
  create_branch: true
`)

	_, err := LoadRepoConfig(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "field git not found") {
		t.Fatalf("unexpected error %q", err)
	}
}
