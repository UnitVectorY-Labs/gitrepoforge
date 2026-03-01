package schema

import (
	"path/filepath"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func TestValidateRepoConfig(t *testing.T) {
	tests := []struct {
		name           string
		repoCfg        *config.RepoConfig
		centralCfg     *config.CentralConfig
		folderName     string
		wantErrCount   int
		wantErrFields  []string
		wantNoErrors   bool
	}{
		{
			name: "valid config with all required inputs",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"language": "go",
					"enabled":  true,
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string", Required: true},
					{Name: "enabled", Type: "boolean", Required: true},
				},
			},
			folderName:   "my-repo",
			wantNoErrors: true,
		},
		{
			name: "missing name",
			repoCfg: &config.RepoConfig{
				Name:   "",
				Inputs: map[string]interface{}{},
			},
			centralCfg:    &config.CentralConfig{},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"name"},
		},
		{
			name: "name mismatch with folder",
			repoCfg: &config.RepoConfig{
				Name:   "wrong-name",
				Inputs: map[string]interface{}{},
			},
			centralCfg:    &config.CentralConfig{},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"name"},
		},
		{
			name: "required input missing",
			repoCfg: &config.RepoConfig{
				Name:   "my-repo",
				Inputs: map[string]interface{}{},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string", Required: true},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.language"},
		},
		{
			name: "unknown input provided",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"unknown_key": "value",
				},
			},
			centralCfg:    &config.CentralConfig{},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.unknown_key"},
		},
		{
			name: "string input with valid enum value",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"language": "go",
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string", Enum: []string{"go", "python", "java"}},
				},
			},
			folderName:   "my-repo",
			wantNoErrors: true,
		},
		{
			name: "string input with invalid enum value",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"language": "rust",
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string", Enum: []string{"go", "python", "java"}},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.language"},
		},
		{
			name: "boolean input with non-boolean value",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"enabled": "yes",
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "enabled", Type: "boolean"},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.enabled"},
		},
		{
			name: "number input with non-number value",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"count": "not-a-number",
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "count", Type: "number"},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.count"},
		},
		{
			name: "list input with non-list value",
			repoCfg: &config.RepoConfig{
				Name: "my-repo",
				Inputs: map[string]interface{}{
					"tags": "not-a-list",
				},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "tags", Type: "list"},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  1,
			wantErrFields: []string{"inputs.tags"},
		},
		{
			name: "all inputs missing when all required",
			repoCfg: &config.RepoConfig{
				Name:   "my-repo",
				Inputs: map[string]interface{}{},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string", Required: true},
					{Name: "enabled", Type: "boolean", Required: true},
					{Name: "count", Type: "number", Required: true},
				},
			},
			folderName:    "my-repo",
			wantErrCount:  3,
			wantErrFields: []string{"inputs.language", "inputs.enabled", "inputs.count"},
		},
		{
			name: "empty inputs map with no required inputs",
			repoCfg: &config.RepoConfig{
				Name:   "my-repo",
				Inputs: map[string]interface{}{},
			},
			centralCfg: &config.CentralConfig{
				Inputs: []config.InputDef{
					{Name: "language", Type: "string"},
					{Name: "enabled", Type: "boolean"},
				},
			},
			folderName:   "my-repo",
			wantNoErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			repoPath := filepath.Join(tmpDir, tt.folderName)

			errs := ValidateRepoConfig(tt.repoCfg, tt.centralCfg, repoPath)

			if tt.wantNoErrors {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %d: %v", len(errs), errs)
				}
				return
			}

			if len(errs) != tt.wantErrCount {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrCount, len(errs), errs)
			}

			errFieldSet := make(map[string]bool)
			for _, e := range errs {
				errFieldSet[e.Field] = true
			}
			for _, field := range tt.wantErrFields {
				if !errFieldSet[field] {
					t.Errorf("expected error on field %q, but not found in %v", field, errs)
				}
			}
		})
	}
}
