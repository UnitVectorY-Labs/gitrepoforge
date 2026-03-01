package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestLoadRootConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string // empty means no file created
		wantErr     bool
		errContains string
		check       func(t *testing.T, cfg *RootConfig)
	}{
		{
			name: "valid full config",
			yaml: `config_repo: my-org/config-repo
default_branch: main
excludes:
  - repo-a
  - repo-b
branch_prefix: "forge/"
create_pr: true
`,
			check: func(t *testing.T, cfg *RootConfig) {
				if cfg.ConfigRepo != "my-org/config-repo" {
					t.Errorf("ConfigRepo = %q, want %q", cfg.ConfigRepo, "my-org/config-repo")
				}
				if cfg.DefaultBranch != "main" {
					t.Errorf("DefaultBranch = %q, want %q", cfg.DefaultBranch, "main")
				}
				if len(cfg.Excludes) != 2 || cfg.Excludes[0] != "repo-a" || cfg.Excludes[1] != "repo-b" {
					t.Errorf("Excludes = %v, want [repo-a repo-b]", cfg.Excludes)
				}
				if cfg.BranchPrefix != "forge/" {
					t.Errorf("BranchPrefix = %q, want %q", cfg.BranchPrefix, "forge/")
				}
				if !cfg.CreatePR {
					t.Error("CreatePR = false, want true")
				}
			},
		},
		{
			name: "default branch_prefix when omitted",
			yaml: `config_repo: org/repo
default_branch: develop
`,
			check: func(t *testing.T, cfg *RootConfig) {
				if cfg.BranchPrefix != "gitrepoforge/" {
					t.Errorf("BranchPrefix = %q, want default %q", cfg.BranchPrefix, "gitrepoforge/")
				}
				if cfg.CreatePR {
					t.Error("CreatePR should default to false")
				}
				if cfg.Excludes != nil {
					t.Errorf("Excludes = %v, want nil", cfg.Excludes)
				}
			},
		},
		{
			name:        "missing file",
			yaml:        "",
			wantErr:     true,
			errContains: "failed to read root config",
		},
		{
			name: "missing config_repo",
			yaml: `default_branch: main
`,
			wantErr:     true,
			errContains: "config_repo is required",
		},
		{
			name: "missing default_branch",
			yaml: `config_repo: org/repo
`,
			wantErr:     true,
			errContains: "default_branch is required",
		},
		{
			name:        "invalid YAML",
			yaml:        ":\n  :\n  - :\n\t- bad",
			wantErr:     true,
			errContains: "failed to parse root config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.yaml != "" {
				writeFile(t, dir, RootConfigFileName, tt.yaml)
			}

			cfg, err := LoadRootConfig(dir)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestResolveConfigRepoPath(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		workspace string
		want      string
	}{
		{
			name:      "relative path",
			repo:      "config-repo",
			workspace: "/workspace",
			want:      filepath.Join("/workspace", "config-repo"),
		},
		{
			name:      "absolute path",
			repo:      "/absolute/config-repo",
			workspace: "/workspace",
			want:      "/absolute/config-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := &RootConfig{ConfigRepo: tt.repo}
			got := rc.ResolveConfigRepoPath(tt.workspace)
			if got != tt.want {
				t.Errorf("ResolveConfigRepoPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadCentralConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		errContains string
		check       func(t *testing.T, cfg *CentralConfig)
	}{
		{
			name: "valid config with inputs and files",
			yaml: `inputs:
  - name: language
    type: string
    required: true
    enum:
      - go
      - python
    default: go
    description: "Primary language"
  - name: enable_ci
    type: boolean
    required: false
files:
  - path: .github/workflows/ci.yml
    action: create
    condition: "inputs.enable_ci"
    template: ci.yml.tmpl
  - path: README.md
    action: patch
    blocks:
      - begin_marker: "<!-- BEGIN MANAGED -->"
        end_marker: "<!-- END MANAGED -->"
        template: readme_block.tmpl
        content: "static content"
`,
			check: func(t *testing.T, cfg *CentralConfig) {
				if len(cfg.Inputs) != 2 {
					t.Fatalf("Inputs length = %d, want 2", len(cfg.Inputs))
				}
				in0 := cfg.Inputs[0]
				if in0.Name != "language" || in0.Type != "string" || !in0.Required {
					t.Errorf("Inputs[0] = %+v", in0)
				}
				if len(in0.Enum) != 2 || in0.Enum[0] != "go" || in0.Enum[1] != "python" {
					t.Errorf("Inputs[0].Enum = %v", in0.Enum)
				}
				if in0.Default != "go" {
					t.Errorf("Inputs[0].Default = %q, want %q", in0.Default, "go")
				}
				if in0.Description != "Primary language" {
					t.Errorf("Inputs[0].Description = %q", in0.Description)
				}

				if len(cfg.Files) != 2 {
					t.Fatalf("Files length = %d, want 2", len(cfg.Files))
				}
				f0 := cfg.Files[0]
				if f0.Path != ".github/workflows/ci.yml" || f0.Action != "create" {
					t.Errorf("Files[0] = %+v", f0)
				}
				if f0.Condition != "inputs.enable_ci" {
					t.Errorf("Files[0].Condition = %q", f0.Condition)
				}
				if f0.Template != "ci.yml.tmpl" {
					t.Errorf("Files[0].Template = %q", f0.Template)
				}

				f1 := cfg.Files[1]
				if len(f1.Blocks) != 1 {
					t.Fatalf("Files[1].Blocks length = %d, want 1", len(f1.Blocks))
				}
				b := f1.Blocks[0]
				if b.BeginMarker != "<!-- BEGIN MANAGED -->" || b.EndMarker != "<!-- END MANAGED -->" {
					t.Errorf("Block markers = %q / %q", b.BeginMarker, b.EndMarker)
				}
				if b.Template != "readme_block.tmpl" || b.Content != "static content" {
					t.Errorf("Block template=%q content=%q", b.Template, b.Content)
				}
			},
		},
		{
			name: "empty config",
			yaml: ``,
			check: func(t *testing.T, cfg *CentralConfig) {
				if cfg.Inputs != nil {
					t.Errorf("Inputs = %v, want nil", cfg.Inputs)
				}
				if cfg.Files != nil {
					t.Errorf("Files = %v, want nil", cfg.Files)
				}
			},
		},
		{
			name: "inputs only",
			yaml: `inputs:
  - name: team
    type: string
`,
			check: func(t *testing.T, cfg *CentralConfig) {
				if len(cfg.Inputs) != 1 {
					t.Fatalf("Inputs length = %d, want 1", len(cfg.Inputs))
				}
				if cfg.Inputs[0].Name != "team" {
					t.Errorf("Inputs[0].Name = %q", cfg.Inputs[0].Name)
				}
				if cfg.Files != nil {
					t.Errorf("Files = %v, want nil", cfg.Files)
				}
			},
		},
		{
			name:        "missing file",
			yaml:        "",
			wantErr:     true,
			errContains: "failed to read central config",
		},
		{
			name:        "invalid YAML",
			yaml:        ":\n  :\n  - :\n\t- bad",
			wantErr:     true,
			errContains: "failed to parse central config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.yaml != "" || (tt.name != "missing file") {
				if tt.yaml != "" || tt.name == "empty config" {
					writeFile(t, dir, CentralConfigFileName, tt.yaml)
				}
			}

			cfg, err := LoadCentralConfig(dir)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadRepoConfig(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		errContains string
		check       func(t *testing.T, cfg *RepoConfig)
	}{
		{
			name: "valid config",
			yaml: `name: my-service
inputs:
  language: go
  enable_ci: true
  replicas: 3
`,
			check: func(t *testing.T, cfg *RepoConfig) {
				if cfg.Name != "my-service" {
					t.Errorf("Name = %q, want %q", cfg.Name, "my-service")
				}
				if cfg.Inputs["language"] != "go" {
					t.Errorf("Inputs[language] = %v", cfg.Inputs["language"])
				}
				if cfg.Inputs["enable_ci"] != true {
					t.Errorf("Inputs[enable_ci] = %v", cfg.Inputs["enable_ci"])
				}
				if cfg.Inputs["replicas"] != 3 {
					t.Errorf("Inputs[replicas] = %v", cfg.Inputs["replicas"])
				}
			},
		},
		{
			name: "minimal config with name only",
			yaml: `name: bare-repo
`,
			check: func(t *testing.T, cfg *RepoConfig) {
				if cfg.Name != "bare-repo" {
					t.Errorf("Name = %q, want %q", cfg.Name, "bare-repo")
				}
				if cfg.Inputs != nil {
					t.Errorf("Inputs = %v, want nil", cfg.Inputs)
				}
			},
		},
		{
			name: "empty file",
			yaml: ``,
			check: func(t *testing.T, cfg *RepoConfig) {
				if cfg.Name != "" {
					t.Errorf("Name = %q, want empty", cfg.Name)
				}
			},
		},
		{
			name:        "missing file",
			yaml:        "",
			wantErr:     true,
			errContains: "failed to read repo config",
		},
		{
			name:        "invalid YAML",
			yaml:        ":\n  :\n  - :\n\t- bad",
			wantErr:     true,
			errContains: "failed to parse repo config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			// Write file for all cases except "missing file"
			if tt.name != "missing file" {
				writeFile(t, dir, RepoConfigFileName, tt.yaml)
			}

			cfg, err := LoadRepoConfig(dir)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestRepoConfigExists(t *testing.T) {
	tests := []struct {
		name       string
		createFile bool
		want       bool
	}{
		{
			name:       "file exists",
			createFile: true,
			want:       true,
		},
		{
			name:       "file does not exist",
			createFile: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.createFile {
				writeFile(t, dir, RepoConfigFileName, "name: test\n")
			}

			got := RepoConfigExists(dir)
			if got != tt.want {
				t.Errorf("RepoConfigExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

