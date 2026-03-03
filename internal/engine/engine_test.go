package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func TestComputeFindings(t *testing.T) {
	tests := []struct {
		name           string
		centralCfg     *config.CentralConfig
		repoCfg        *config.RepoConfig
		setupRepo      func(t *testing.T, repoPath string)
		wantFindings   int
		wantOperation  string
		wantMessage    string
		wantExpected   string
		wantActual     string
		wantNoFindings bool
	}{
		{
			name: "create rule - file does not exist",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "README.md", Mode: "create", Template: "hello world"},
				},
			},
			repoCfg:       &config.RepoConfig{Name: "test-repo"},
			setupRepo:     func(t *testing.T, repoPath string) {},
			wantFindings:  1,
			wantOperation: "create",
			wantMessage:   "file does not exist but should",
			wantExpected:  "hello world",
		},
		{
			name: "create rule - file exists but content differs",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "README.md", Mode: "create", Template: "expected content"},
				},
			},
			repoCfg: &config.RepoConfig{Name: "test-repo"},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("actual content"), 0644)
			},
			wantFindings:  1,
			wantOperation: "update",
			wantMessage:   "file content differs from expected",
			wantExpected:  "expected content",
			wantActual:    "actual content",
		},
		{
			name: "create rule - file matches expected",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "README.md", Mode: "create", Template: "matching content"},
				},
			},
			repoCfg: &config.RepoConfig{Name: "test-repo"},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("matching content"), 0644)
			},
			wantNoFindings: true,
		},
		{
			name: "delete rule - file exists",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "obsolete.txt", Mode: "delete"},
				},
			},
			repoCfg: &config.RepoConfig{Name: "test-repo"},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "obsolete.txt"), []byte("old data"), 0644)
			},
			wantFindings:  1,
			wantOperation: "delete",
			wantMessage:   "file exists but should not",
		},
		{
			name: "delete rule - file does not exist",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "obsolete.txt", Mode: "delete"},
				},
			},
			repoCfg:        &config.RepoConfig{Name: "test-repo"},
			setupRepo:      func(t *testing.T, repoPath string) {},
			wantNoFindings: true,
		},
		{
			name: "partial rule - managed block differs",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{
						Path: "config.yaml",
						Mode: "partial",
						Blocks: []config.BlockRule{
							{
								BeginMarker: "# BEGIN MANAGED",
								EndMarker:   "# END MANAGED",
								Template:    "new block content",
							},
						},
					},
				},
			},
			repoCfg: &config.RepoConfig{Name: "test-repo"},
			setupRepo: func(t *testing.T, repoPath string) {
				content := "header\n# BEGIN MANAGED\nold block content\n# END MANAGED\nfooter\n"
				os.WriteFile(filepath.Join(repoPath, "config.yaml"), []byte(content), 0644)
			},
			wantFindings:  1,
			wantOperation: "block_replace",
			wantMessage:   "managed blocks differ from expected",
			wantExpected:  "header\n# BEGIN MANAGED\nnew block content\n# END MANAGED\nfooter\n",
			wantActual:    "header\n# BEGIN MANAGED\nold block content\n# END MANAGED\nfooter\n",
		},
		{
			name: "condition evaluates to false - no findings",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "README.md", Mode: "create", Template: "hello", Condition: "enabled"},
				},
			},
			repoCfg: &config.RepoConfig{
				Name:   "test-repo",
				Inputs: map[string]interface{}{"enabled": false},
			},
			setupRepo:      func(t *testing.T, repoPath string) {},
			wantNoFindings: true,
		},
		{
			name: "template rendering with inputs",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "greeting.txt", Mode: "create", Template: "Hello {{.Name}} - {{.Inputs.lang}}"},
				},
			},
			repoCfg: &config.RepoConfig{
				Name:   "my-project",
				Inputs: map[string]interface{}{"lang": "Go"},
			},
			setupRepo:     func(t *testing.T, repoPath string) {},
			wantFindings:  1,
			wantOperation: "create",
			wantExpected:  "Hello my-project - Go",
		},
		{
			name: "condition with equality check",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "go.mod", Mode: "create", Template: "module example", Condition: `language == "go"`},
				},
			},
			repoCfg: &config.RepoConfig{
				Name:   "test-repo",
				Inputs: map[string]interface{}{"language": "go"},
			},
			setupRepo:     func(t *testing.T, repoPath string) {},
			wantFindings:  1,
			wantOperation: "create",
			wantExpected:  "module example",
		},
		{
			name: "condition with inequality check - skips",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "go.mod", Mode: "create", Template: "module example", Condition: `language == "python"`},
				},
			},
			repoCfg: &config.RepoConfig{
				Name:   "test-repo",
				Inputs: map[string]interface{}{"language": "go"},
			},
			setupRepo:      func(t *testing.T, repoPath string) {},
			wantNoFindings: true,
		},
		{
			name: "partial rule - file does not exist creates managed blocks",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{
						Path: "new-partial.txt",
						Mode: "partial",
						Blocks: []config.BlockRule{
							{
								BeginMarker: "# BEGIN",
								EndMarker:   "# END",
								Template:    "managed line",
							},
						},
					},
				},
			},
			repoCfg:       &config.RepoConfig{Name: "test-repo"},
			setupRepo:     func(t *testing.T, repoPath string) {},
			wantFindings:  1,
			wantOperation: "create",
			wantExpected:  "# BEGIN\nmanaged line\n# END",
		},
		{
			name: "multiple rules - mixed results",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{Path: "exists.txt", Mode: "create", Template: "ok"},
					{Path: "missing.txt", Mode: "create", Template: "needed"},
				},
			},
			repoCfg: &config.RepoConfig{Name: "test-repo"},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "exists.txt"), []byte("ok"), 0644)
			},
			wantFindings:  1,
			wantOperation: "create",
			wantMessage:   "file does not exist but should",
		},
		{
			name: "template with conditional section",
			centralCfg: &config.CentralConfig{
				Files: []config.FileRule{
					{
						Path: "config.txt",
						Mode: "create",
						Template: `base config
{{if .Inputs.debug}}debug: true
{{end}}done`,
					},
				},
			},
			repoCfg: &config.RepoConfig{
				Name:   "test-repo",
				Inputs: map[string]interface{}{"debug": true},
			},
			setupRepo:     func(t *testing.T, repoPath string) {},
			wantFindings:  1,
			wantOperation: "create",
			wantExpected:  "base config\ndebug: true\ndone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := t.TempDir()

			tt.setupRepo(t, repoPath)

			findings, err := ComputeFindings(tt.repoCfg, tt.centralCfg, repoPath)
			if err != nil {
				t.Fatalf("ComputeFindings returned error: %v", err)
			}

			if tt.wantNoFindings {
				if len(findings) != 0 {
					t.Fatalf("expected no findings, got %d: %+v", len(findings), findings)
				}
				return
			}

			if len(findings) != tt.wantFindings {
				t.Fatalf("expected %d findings, got %d: %+v", tt.wantFindings, len(findings), findings)
			}

			f := findings[0]
			if tt.wantOperation != "" && f.Operation != tt.wantOperation {
				t.Errorf("expected operation %q, got %q", tt.wantOperation, f.Operation)
			}
			if tt.wantMessage != "" && f.Message != tt.wantMessage {
				t.Errorf("expected message %q, got %q", tt.wantMessage, f.Message)
			}
			if tt.wantExpected != "" && f.Expected != tt.wantExpected {
				t.Errorf("expected Expected %q, got %q", tt.wantExpected, f.Expected)
			}
			if tt.wantActual != "" && f.Actual != tt.wantActual {
				t.Errorf("expected Actual %q, got %q", tt.wantActual, f.Actual)
			}
		})
	}
}

func TestApplyFindings(t *testing.T) {
	tests := []struct {
		name      string
		findings  []Finding
		setupRepo func(t *testing.T, repoPath string)
		verify    func(t *testing.T, repoPath string)
	}{
		{
			name: "create new file",
			findings: []Finding{
				{FilePath: "newfile.txt", Operation: "create", Expected: "new content"},
			},
			setupRepo: func(t *testing.T, repoPath string) {},
			verify: func(t *testing.T, repoPath string) {
				data, err := os.ReadFile(filepath.Join(repoPath, "newfile.txt"))
				if err != nil {
					t.Fatalf("file not created: %v", err)
				}
				if string(data) != "new content" {
					t.Errorf("expected %q, got %q", "new content", string(data))
				}
			},
		},
		{
			name: "update existing file",
			findings: []Finding{
				{FilePath: "existing.txt", Operation: "update", Expected: "updated content"},
			},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "existing.txt"), []byte("old content"), 0644)
			},
			verify: func(t *testing.T, repoPath string) {
				data, err := os.ReadFile(filepath.Join(repoPath, "existing.txt"))
				if err != nil {
					t.Fatalf("file not found: %v", err)
				}
				if string(data) != "updated content" {
					t.Errorf("expected %q, got %q", "updated content", string(data))
				}
			},
		},
		{
			name: "delete file",
			findings: []Finding{
				{FilePath: "todelete.txt", Operation: "delete"},
			},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "todelete.txt"), []byte("will be deleted"), 0644)
			},
			verify: func(t *testing.T, repoPath string) {
				if _, err := os.Stat(filepath.Join(repoPath, "todelete.txt")); !os.IsNotExist(err) {
					t.Errorf("expected file to be deleted, but it still exists")
				}
			},
		},
		{
			name: "block_replace operation",
			findings: []Finding{
				{FilePath: "partial.txt", Operation: "block_replace", Expected: "replaced block content"},
			},
			setupRepo: func(t *testing.T, repoPath string) {
				os.WriteFile(filepath.Join(repoPath, "partial.txt"), []byte("original"), 0644)
			},
			verify: func(t *testing.T, repoPath string) {
				data, err := os.ReadFile(filepath.Join(repoPath, "partial.txt"))
				if err != nil {
					t.Fatalf("file not found: %v", err)
				}
				if string(data) != "replaced block content" {
					t.Errorf("expected %q, got %q", "replaced block content", string(data))
				}
			},
		},
		{
			name: "create file in nested directory",
			findings: []Finding{
				{FilePath: "sub/dir/deep.txt", Operation: "create", Expected: "deep content"},
			},
			setupRepo: func(t *testing.T, repoPath string) {},
			verify: func(t *testing.T, repoPath string) {
				data, err := os.ReadFile(filepath.Join(repoPath, "sub", "dir", "deep.txt"))
				if err != nil {
					t.Fatalf("file not created: %v", err)
				}
				if string(data) != "deep content" {
					t.Errorf("expected %q, got %q", "deep content", string(data))
				}
			},
		},
		{
			name: "delete non-existing file does not error",
			findings: []Finding{
				{FilePath: "nonexistent.txt", Operation: "delete"},
			},
			setupRepo: func(t *testing.T, repoPath string) {},
			verify: func(t *testing.T, repoPath string) {
				// Just verifying no error was returned (handled by test runner)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := t.TempDir()
			tt.setupRepo(t, repoPath)

			err := ApplyFindings(tt.findings, repoPath)
			if err != nil {
				t.Fatalf("ApplyFindings returned error: %v", err)
			}

			tt.verify(t, repoPath)
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		data      TemplateData
		want      bool
	}{
		{
			name:      "empty condition returns true",
			condition: "",
			data:      TemplateData{},
			want:      true,
		},
		{
			name:      "key == value matching",
			condition: "lang == Go",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      true,
		},
		{
			name:      "key == value not matching",
			condition: "lang == Python",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      false,
		},
		{
			name:      "key == value with quoted value",
			condition: `lang == "Go"`,
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      true,
		},
		{
			name:      "key == value with missing key",
			condition: "missing == value",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      false,
		},
		{
			name:      "key != value matching (values differ)",
			condition: "lang != Python",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      true,
		},
		{
			name:      "key != value not matching (values same)",
			condition: "lang != Go",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      false,
		},
		{
			name:      "key != value with missing key",
			condition: "missing != value",
			data:      TemplateData{Inputs: map[string]interface{}{"lang": "Go"}},
			want:      true,
		},
		{
			name:      "boolean input true",
			condition: "enabled",
			data:      TemplateData{Inputs: map[string]interface{}{"enabled": true}},
			want:      true,
		},
		{
			name:      "boolean input false",
			condition: "enabled",
			data:      TemplateData{Inputs: map[string]interface{}{"enabled": false}},
			want:      false,
		},
		{
			name:      "string input non-empty is true",
			condition: "name",
			data:      TemplateData{Inputs: map[string]interface{}{"name": "hello"}},
			want:      true,
		},
		{
			name:      "string input empty is false",
			condition: "name",
			data:      TemplateData{Inputs: map[string]interface{}{"name": ""}},
			want:      false,
		},
		{
			name:      "string input 'false' is false",
			condition: "flag",
			data:      TemplateData{Inputs: map[string]interface{}{"flag": "false"}},
			want:      false,
		},
		{
			name:      "non-bool non-string input is true",
			condition: "count",
			data:      TemplateData{Inputs: map[string]interface{}{"count": 42}},
			want:      true,
		},
		{
			name:      "missing boolean input returns false",
			condition: "missing",
			data:      TemplateData{Inputs: map[string]interface{}{}},
			want:      false,
		},
		{
			name:      "nil inputs with boolean condition returns false",
			condition: "enabled",
			data:      TemplateData{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateCondition(tt.condition, tt.data)
			if got != tt.want {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}

func TestReplaceBlock(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		beginMarker string
		endMarker   string
		newBlock    string
		want        string
	}{
		{
			name:        "markers present - content replaced",
			content:     "header\n# BEGIN\nold content\n# END\nfooter",
			beginMarker: "# BEGIN",
			endMarker:   "# END",
			newBlock:    "new content",
			want:        "header\n# BEGIN\nnew content\n# END\nfooter",
		},
		{
			name:        "markers not present - content appended",
			content:     "existing content",
			beginMarker: "# BEGIN",
			endMarker:   "# END",
			newBlock:    "appended content",
			want:        "existing content\n# BEGIN\nappended content\n# END\n",
		},
		{
			name:        "only begin marker present - content appended",
			content:     "line1\n# BEGIN\nline2",
			beginMarker: "# BEGIN",
			endMarker:   "# END",
			newBlock:    "new stuff",
			want:        "line1\n# BEGIN\nline2\n# BEGIN\nnew stuff\n# END\n",
		},
		{
			name:        "empty content with markers",
			content:     "# BEGIN\n# END",
			beginMarker: "# BEGIN",
			endMarker:   "# END",
			newBlock:    "inserted",
			want:        "# BEGIN\ninserted\n# END",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceBlock(tt.content, tt.beginMarker, tt.endMarker, tt.newBlock)
			if got != tt.want {
				t.Errorf("replaceBlock() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
