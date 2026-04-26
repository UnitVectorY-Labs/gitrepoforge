package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func withoutManagedFilesManifest(findings []Finding) []Finding {
	cloned := append([]Finding(nil), findings...)
	return stripManagedFilesManifest(cloned)
}

func requireManagedFilesManifestFinding(t *testing.T, findings []Finding) Finding {
	t.Helper()

	finding := managedFilesManifestFinding(findings)
	if finding == nil {
		t.Fatalf("expected %s finding", config.ManagedFilesManifestName)
	}
	return *finding
}

func TestComputeFindingsSelectsMatchingTemplate(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n{{.Name}}\n")
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
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Expected != "MIT License\n{{.Name}}\n" {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, "MIT License\n{{.Name}}\n")
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

func TestComputeFindingsAppliesDefaultsBeforeSelectingTemplate(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n")

	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Required: true, Default: "mit", HasDefault: true},
		},
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
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if repoCfg.Config["license"] != "mit" {
		t.Fatalf("Config[license] = %v, want %q", repoCfg.Config["license"], "mit")
	}
}

func TestComputeFindingsEvaluatesTemplateWhenRequested(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/justfile.tmpl", `# Commands for {{.Name}}
default:
  @just --list

{{- if eq .Config.language "go" }}
# Build {{.Name}} with Go
build:
  go build ./...
{{- end }}
{{- if eq .Config.language "java" }}
# Build {{.Name}} with Maven
build:
  mvn package
{{- end }}
`)

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "justfile",
				Templates: []config.TemplateRef{
					{
						Condition:    "justfile",
						Template:     "justfile.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "justfile.tmpl"),
					},
					{
						Absent: true,
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"justfile": true,
			"language": "go",
		},
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	want := "# Commands for example-repo\ndefault:\n  @just --list\n# Build example-repo with Go\nbuild:\n  go build ./...\n"
	if findings[0].Expected != want {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, want)
	}
}

func TestComputeFindingsSupportsStrictDoubleBracketTemplateMode(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/.github/workflows/ci.yml.tmpl", `name: ci
jobs:
  test:
    steps:
      - uses: actions/cache@v4
        with:
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
{{- if eq .Config.codecov true }}
      - uses: codecov/codecov-action@v4
{{- end }}
`)

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: filepath.Join(".github", "workflows", "ci.yml"),
				Templates: []config.TemplateRef{
					{
						Template:     filepath.Join(".github", "workflows", "ci.yml.tmpl"),
						Evaluate:     true,
						TemplateMode: config.TemplateModeDoubleBracketStrict,
						ResolvedPath: filepath.Join(configRepo, "templates", ".github", "workflows", "ci.yml.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"codecov": true,
		},
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	want := "name: ci\njobs:\n  test:\n    steps:\n      - uses: actions/cache@v4\n        with:\n          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}\n      - uses: codecov/codecov-action@v4\n"
	if findings[0].Expected != want {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, want)
	}
}

func TestComputeFindingsQuoteHelperEscapesTemplateStrings(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/action.yml.tmpl", `name: test
description: {{ .Config.description | quote_double }}
summary: {{ .Config.summary | quote_single }}
go-version: {{ .Config.versions.go | quote_double }}
`)

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "action.yml",
				Templates: []config.TemplateRef{
					{
						Template:     "action.yml.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "action.yml.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"description": "It's \"quoted\"\nand escaped",
			"summary":     `It's "quoted"`,
			"versions": map[string]interface{}{
				"go": "1.26.0",
			},
		},
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	want := "name: test\ndescription: \"It's \\\"quoted\\\"\\nand escaped\"\nsummary: 'It''s \"quoted\"'\ngo-version: \"1.26.0\"\n"
	if findings[0].Expected != want {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, want)
	}
}

func TestComputeFindingsQuoteHelperRejectsUnknownHelper(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/action.yml.tmpl", `description: {{ .Config.description | quote_backticks }}
`)

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "action.yml",
				Templates: []config.TemplateRef{
					{
						Template:     "action.yml.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "action.yml.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"description": "quoted",
		},
	}

	_, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `function "quote_backticks" not defined`) {
		t.Fatalf("error = %q, want unknown helper message", err.Error())
	}
}

func TestComputeFindingsMaterializesNestedDefaultsForOptionalObjects(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/docs/enabled.tmpl", `{{ .Config.docs.enabled }}`)

	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{
				Name: "docs",
				Type: "object",
				Attributes: []config.ConfigDefinition{
					{Name: "enabled", Type: "boolean", Default: true, HasDefault: true},
				},
			},
		},
		Files: []config.FileRule{
			{
				Path: "docs/enabled.txt",
				Templates: []config.TemplateRef{
					{
						Condition:    "docs.enabled",
						Template:     "docs/enabled.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "docs", "enabled.tmpl"),
					},
					{
						Absent: true,
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Expected != "true" {
		t.Fatalf("Expected = %q, want %q", findings[0].Expected, "true")
	}

	docs, ok := repoCfg.Config["docs"].(map[string]interface{})
	if !ok {
		t.Fatalf("Config[docs] has unexpected type %T", repoCfg.Config["docs"])
	}
	if docs["enabled"] != true {
		t.Fatalf("Config[docs][enabled] = %v, want true", docs["enabled"])
	}
}

func TestComputeFindingsExistsConditionUsesExplicitConfig(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/docs/CNAME.tmpl", `{{ .Config.docs.domain }}`)

	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{
				Name: "docs",
				Type: "object",
				Attributes: []config.ConfigDefinition{
					{Name: "enabled", Type: "boolean", Default: true, HasDefault: true},
					{Name: "domain", Type: "string", Default: "docs.default.example.com", HasDefault: true},
				},
			},
		},
		Files: []config.FileRule{
			{
				Path: "docs/CNAME",
				Templates: []config.TemplateRef{
					{
						Condition:    "docs.enabled && exists docs.domain",
						Template:     "docs/CNAME.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "docs", "CNAME.tmpl"),
					},
					{
						Absent: true,
					},
				},
			},
		},
	}

	t.Run("defaulted value does not count as existing", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
		}

		findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
		if err != nil {
			t.Fatalf("ComputeFindings returned error: %v", err)
		}
		findings = withoutManagedFilesManifest(findings)
		if len(findings) != 0 {
			t.Fatalf("expected 0 findings, got %d", len(findings))
		}

		docs, ok := repoCfg.Config["docs"].(map[string]interface{})
		if !ok {
			t.Fatalf("Config[docs] has unexpected type %T", repoCfg.Config["docs"])
		}
		if docs["domain"] != "docs.default.example.com" {
			t.Fatalf("Config[docs][domain] = %v, want %q", docs["domain"], "docs.default.example.com")
		}
	})

	t.Run("explicit value counts as existing", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config: map[string]interface{}{
				"docs": map[string]interface{}{
					"enabled": true,
					"domain":  "docs.example.com",
				},
			},
		}

		findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
		if err != nil {
			t.Fatalf("ComputeFindings returned error: %v", err)
		}
		findings = withoutManagedFilesManifest(findings)
		if len(findings) != 1 {
			t.Fatalf("expected 1 finding, got %d", len(findings))
		}
		if findings[0].Expected != "docs.example.com" {
			t.Fatalf("Expected = %q, want %q", findings[0].Expected, "docs.example.com")
		}
	})

	t.Run("explicit value does not match when boolean condition is false", func(t *testing.T) {
		repoCfg := &config.RepoConfig{
			Name:          "example-repo",
			DefaultBranch: "main",
			Config: map[string]interface{}{
				"docs": map[string]interface{}{
					"enabled": false,
					"domain":  "docs.example.com",
				},
			},
		}

		findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
		if err != nil {
			t.Fatalf("ComputeFindings returned error: %v", err)
		}
		findings = withoutManagedFilesManifest(findings)
		if len(findings) != 0 {
			t.Fatalf("expected 0 findings, got %d", len(findings))
		}
	})
}

func TestComputeFindingsDeletesFileWhenAbsentCandidateMatches(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/justfile.tmpl", "ignored")
	repoPath := t.TempDir()
	writeTestFile(t, repoPath, "justfile", "old content\n")

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "justfile",
				Templates: []config.TemplateRef{
					{
						Condition:    "justfile",
						Template:     "justfile.tmpl",
						Evaluate:     true,
						ResolvedPath: filepath.Join(configRepo, "templates", "justfile.tmpl"),
					},
					{
						Absent: true,
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
		Config: map[string]interface{}{
			"justfile": false,
		},
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, repoPath)
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	findings = withoutManagedFilesManifest(findings)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Operation != "delete" {
		t.Fatalf("Operation = %q, want %q", findings[0].Operation, "delete")
	}
}

func TestComputeFindingsAddsManagedFilesManifest(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n")
	writeTestFile(t, configRepo, "templates/README.md.tmpl", `{{ section start=start_of_file end=contains("<!-- END MANAGED -->") }}
# Example Repo
<!-- END MANAGED -->
{{ end }}
`)
	writeTestFile(t, configRepo, "templates/NOTICE.tmpl", `{{ bootstrap }}
Bootstrap only
{{ end }}
`)

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "LICENSE",
				Templates: []config.TemplateRef{
					{
						Template:     "licenses/mit.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "licenses", "mit.tmpl"),
					},
				},
			},
			{
				Path: "README.md",
				Templates: []config.TemplateRef{
					{
						Template:     "README.md.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "README.md.tmpl"),
					},
				},
			},
			{
				Path: "NOTICE",
				Templates: []config.TemplateRef{
					{
						Template:     "NOTICE.tmpl",
						ResolvedPath: filepath.Join(configRepo, "templates", "NOTICE.tmpl"),
					},
				},
			},
		},
	}
	repoCfg := &config.RepoConfig{
		Name:          "example-repo",
		DefaultBranch: "main",
	}

	findings, err := ComputeFindings(repoCfg, centralCfg, t.TempDir())
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}

	manifestFinding := requireManagedFilesManifestFinding(t, findings)
	if manifestFinding.Operation != "create" {
		t.Fatalf("Operation = %q, want %q", manifestFinding.Operation, "create")
	}

	manifest, err := parseManagedFilesManifest(manifestFinding.Expected)
	if err != nil {
		t.Fatalf("parseManagedFilesManifest returned error: %v", err)
	}
	gotPaths := make([]string, 0, len(manifest.ManagedFiles))
	for _, entry := range manifest.ManagedFiles {
		gotPaths = append(gotPaths, entry.Path)
	}
	if want := []string{
		".managedfiles.yaml",
		"LICENSE",
		"README.md",
	}; !reflect.DeepEqual(gotPaths, want) {
		t.Fatalf("managed file paths = %v, want %v", gotPaths, want)
	}

	for _, entry := range manifest.ManagedFiles {
		if entry.Path != "README.md" {
			continue
		}
		if entry.Management != "sections" {
			t.Fatalf("README.md management = %q, want %q", entry.Management, "sections")
		}
		if len(entry.Sections) != 1 {
			t.Fatalf("README.md sections length = %d, want 1", len(entry.Sections))
		}
		if entry.Sections[0].Start != "start_of_file" {
			t.Fatalf("README.md section start = %q, want %q", entry.Sections[0].Start, "start_of_file")
		}
		if entry.Sections[0].End != `contains("<!-- END MANAGED -->")` {
			t.Fatalf("README.md section end = %q, want %q", entry.Sections[0].End, `contains("<!-- END MANAGED -->")`)
		}
		return
	}

	t.Fatal("expected README.md manifest entry")
}

func TestComputeFindingsTreatsManagedFilesManifestAsCleanWhenCurrent(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/licenses/mit.tmpl", "MIT License\n")

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: "LICENSE",
				Templates: []config.TemplateRef{
					{
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
	}
	repoPath := t.TempDir()
	writeTestFile(t, repoPath, "LICENSE", "MIT License\n")

	manifestContent, err := renderManagedFilesManifest(TemplateData{
		Name:          repoCfg.Name,
		DefaultBranch: repoCfg.DefaultBranch,
		Config:        map[string]interface{}{},
	}, centralCfg)
	if err != nil {
		t.Fatalf("renderManagedFilesManifest returned error: %v", err)
	}
	writeTestFile(t, repoPath, config.ManagedFilesManifestName, manifestContent)

	findings, err := ComputeFindings(repoCfg, centralCfg, repoPath)
	if err != nil {
		t.Fatalf("ComputeFindings returned error: %v", err)
	}
	if managedFilesManifestFinding(findings) != nil {
		t.Fatalf("expected %s to be clean", config.ManagedFilesManifestName)
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
		name           string
		condition      string
		values         map[string]interface{}
		providedValues map[string]interface{}
		want           bool
		wantErr        bool
	}{
		{name: "empty condition matches", condition: "", want: true},
		{name: "boolean key true", condition: "enabled", values: map[string]interface{}{"enabled": true}, want: true},
		{name: "boolean key false", condition: "!enabled", values: map[string]interface{}{"enabled": false}, want: true},
		{name: "string equality", condition: `license == "mit"`, values: map[string]interface{}{"license": "mit"}, want: true},
		{name: "string inequality", condition: `license != "apache-2.0"`, values: map[string]interface{}{"license": "mit"}, want: true},
		{name: "nested boolean key", condition: "docs.enabled", values: map[string]interface{}{"docs": map[string]interface{}{"enabled": true}}, want: true},
		{name: "nested string equality", condition: `docs.domain == "foo.example.com"`, values: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "exists key present", condition: "exists docs.domain", providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "exists key missing", condition: "exists docs.domain", providedValues: map[string]interface{}{}, want: false},
		{name: "not exists key missing", condition: "!exists docs.domain", providedValues: map[string]interface{}{}, want: true},
		{name: "exists ignores defaulted value", condition: "exists docs.domain", values: map[string]interface{}{"docs": map[string]interface{}{"domain": "default.example.com"}}, providedValues: map[string]interface{}{}, want: false},
		{name: "and expression", condition: "docs.enabled && exists docs.domain", values: map[string]interface{}{"docs": map[string]interface{}{"enabled": true}}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "and expression false", condition: "docs.enabled && exists docs.domain", values: map[string]interface{}{"docs": map[string]interface{}{"enabled": false}}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: false},
		{name: "or expression", condition: "docs.enabled || exists docs.domain", values: map[string]interface{}{"docs": map[string]interface{}{"enabled": false}}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "operator precedence", condition: "enabled || other && exists docs.domain", values: map[string]interface{}{"enabled": false, "other": true}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "grouped expression", condition: "(enabled || other) && exists docs.domain", values: map[string]interface{}{"enabled": false, "other": true}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, want: true},
		{name: "missing key equality", condition: `license == "mit"`, values: map[string]interface{}{}, want: false},
		{name: "bare non boolean is invalid", condition: "license", values: map[string]interface{}{"license": "mit"}, wantErr: true},
		{name: "invalid exists condition", condition: "exists", wantErr: true},
		{name: "missing closing parenthesis", condition: "(enabled && exists docs.domain", values: map[string]interface{}{"enabled": true}, providedValues: map[string]interface{}{"docs": map[string]interface{}{"domain": "foo.example.com"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCondition(tt.condition, tt.values, tt.providedValues)
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
