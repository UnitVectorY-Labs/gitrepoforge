package schema

import (
	"path/filepath"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func TestValidateRepoConfig(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Required: true, Enum: []string{"mit", "apache-2.0"}},
			{Name: "enabled", Type: "boolean"},
		},
	}

	t.Run("valid config", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config: map[string]interface{}{
				"license": "mit",
				"enabled": true,
			},
		}

		errs := ValidateRepoConfig(repoCfg, centralCfg, filepath.Join(t.TempDir(), "example-repo"))
		if len(errs) != 0 {
			t.Fatalf("expected no validation errors, got %v", errs)
		}
	})

	t.Run("missing required config value", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config:        map[string]interface{}{},
		}

		errs := ValidateRepoConfig(repoCfg, centralCfg, filepath.Join(t.TempDir(), "example-repo"))
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "config.license" {
			t.Fatalf("Field = %q, want %q", errs[0].Field, "config.license")
		}
	})

	t.Run("unknown config value", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config: map[string]interface{}{
				"license": "mit",
				"other":   "x",
			},
		}

		errs := ValidateRepoConfig(repoCfg, centralCfg, filepath.Join(t.TempDir(), "example-repo"))
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "config.other" {
			t.Fatalf("Field = %q, want %q", errs[0].Field, "config.other")
		}
	})

	t.Run("enum mismatch", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config: map[string]interface{}{
				"license": "gpl-3.0",
			},
		}

		errs := ValidateRepoConfig(repoCfg, centralCfg, filepath.Join(t.TempDir(), "example-repo"))
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "config.license" {
			t.Fatalf("Field = %q, want %q", errs[0].Field, "config.license")
		}
	})

	t.Run("missing default branch", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name: "example-repo",
			Config: map[string]interface{}{
				"license": "mit",
			},
		}

		errs := ValidateRepoConfig(repoCfg, centralCfg, filepath.Join(t.TempDir(), "example-repo"))
		if len(errs) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(errs), errs)
		}
		if errs[0].Field != "default_branch" {
			t.Fatalf("Field = %q, want %q", errs[0].Field, "default_branch")
		}
	})
}
