package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func writeTestFile(t *testing.T, dir, relPath, content string) string {
	t.Helper()

	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
	return fullPath
}

func TestComputeFindingsSelectsMatchingTemplate(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n")
	writeTestFile(t, configRepo, "templates/licenses/apache-2.0.tmpl", "Apache License 2.0\n")

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "LICENSE",
				Templates: []config.TemplateRef{
					{
						Condition:    `license == "mit"`,
						Template:     "licenses/mit.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "licenses", "mit.tmpl"),
					},
					{
						Condition:    `license == "apache-2.0"`,
						Template:     "licenses/apache-2.0.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "licenses", "apache-2.0.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"license": "mit",
		},
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Expected != "MIT License\n" {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, "MIT License\n")
	}
}

func TestComputeFindingsReturnsErrorWhenNoTemplateMatches(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n")

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "LICENSE",
				Templates: []config.TemplateRef{
					{
						Condition:    `license == "mit"`,
						Template:     "licenses/mit.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "licenses", "mit.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"license": "apache-2.0",
		},
	}

	_, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApplyFindings(t *testing.T) {
	repoPath := t.TempDir()
	findings := []Finding{
		{FilePath: "LICENSE", Operation: "create", Expected: "MIT License\n"},
		{FilePath: "obsolete.txt", Operation: "delete"},
	}
	writeTestFile(t, repoPath, "obsolete.txt", "remove me")

	if err := ApplyFindings(findings, repoPath); err != nil {
		t.Fatalf("ApplyFindings returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoPath, "LICENSE"))
	if err != nil {
		t.Fatalf("failed to read created LICENSE file: %v", err)
	}
	if string(data) != "MIT License\n" {
		t.Fatalf("LICENSE content = %q, want %q", string(data), "MIT License\n")
	}
	if _, err := os.Stat(filepath.Join(repoPath, "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected obsolete.txt to be deleted")
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		values    map[string]interface{}
		want      bool
		wantErr   bool
	}{
		{name: "empty condition matches", condition: "", want: true},
		{name: "boolean key true", condition: "enabled", values: map[string]interface{}{"enabled": true}, want: true},
		{name: "boolean key false", condition: "!enabled", values: map[string]interface{}{"enabled": false}, want: true},
		{name: "string equality", condition: `license == "mit"`, values: map[string]interface{}{"license": "mit"}, want: true},
		{name: "string inequality", condition: `license != "apache-2.0"`, values: map[string]interface{}{"license": "mit"}, want: true},
		{name: "missing key equality", condition: `license == "mit"`, values: map[string]interface{}{}, want: false},
		{name: "bare non boolean is invalid", condition: "license", values: map[string]interface{}{"license": "mit"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("EvaluateCondition returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("EvaluateCondition(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}
