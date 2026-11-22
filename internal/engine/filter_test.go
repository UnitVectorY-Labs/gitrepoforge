package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func TestNewFileFilterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		include []string
		exclude []string
		wantErr string
	}{
		{
			name: "empty filter is valid",
		},
		{
			name:    "valid globs",
			include: []string{".github/workflows/*.yml", "LICENSE", "./docs/**"},
			exclude: []string{"**/*.bak", "justfile"},
		},
		{
			name:    "empty include pattern rejected",
			include: []string{"   "},
			wantErr: "--file pattern must not be empty",
		},
		{
			name:    "empty exclude pattern rejected",
			exclude: []string{""},
			wantErr: "--exclude-file pattern must not be empty",
		},
		{
			name:    "dot pattern rejected",
			include: []string{"./"},
			wantErr: "--file pattern must not be empty",
		},
		{
			name:    "bad glob syntax rejected",
			include: []string{"docs/[unclosed"},
			wantErr: "invalid --file pattern",
		},
		{
			name:    "bad exclude glob syntax rejected",
			exclude: []string{"docs/[unclosed"},
			wantErr: "invalid --exclude-file pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewFileFilter(tt.include, tt.exclude)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("NewFileFilter returned unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestFileFilterMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		include []string
		exclude []string
		path    string
		want    bool
	}{
		{
			name: "unfiltered matches any path",
			path: ".github/workflows/ci.yml",
			want: true,
		},
		{
			name:    "exact match",
			include: []string{"LICENSE"},
			path:    "LICENSE",
			want:    true,
		},
		{
			name:    "exact mismatch",
			include: []string{"LICENSE"},
			path:    "docs/LICENSE",
			want:    false,
		},
		{
			name:    "leading dot-slash normalized",
			include: []string{"./.github/workflows/ci.yml"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "single star does not cross slash",
			include: []string{".github/workflows/*.yml"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "single star rejects nested path",
			include: []string{".github/workflows/*.yml"},
			path:    ".github/workflows/sub/ci.yml",
			want:    false,
		},
		{
			name:    "trailing doublestar matches nested files",
			include: []string{".github/**"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "trailing doublestar matches direct child",
			include: []string{".github/**"},
			path:    ".github/dependabot.yml",
			want:    true,
		},
		{
			name:    "trailing doublestar does not match prefix itself",
			include: []string{".github/**"},
			path:    ".github",
			want:    false,
		},
		{
			name:    "leading doublestar matches root file",
			include: []string{"**/*.yml"},
			path:    "ci.yml",
			want:    true,
		},
		{
			name:    "leading doublestar matches deeply nested file",
			include: []string{"**/*.yml"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "middle doublestar matches zero directories",
			include: []string{".github/**/ci.yml"},
			path:    ".github/ci.yml",
			want:    true,
		},
		{
			name:    "middle doublestar matches nested directories",
			include: []string{".github/**/ci.yml"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "standalone doublestar matches everything",
			include: []string{"**"},
			path:    "a/b/c.txt",
			want:    true,
		},
		{
			name:    "multiple includes act as union",
			include: []string{"LICENSE", "README.md"},
			path:    "README.md",
			want:    true,
		},
		{
			name:    "exclude overrides include",
			include: []string{".github/workflows/*.yml"},
			exclude: []string{".github/workflows/release*.yml"},
			path:    ".github/workflows/release-go.yml",
			want:    false,
		},
		{
			name:    "exclude keeps non-matching included file",
			include: []string{".github/workflows/*.yml"},
			exclude: []string{".github/workflows/release*.yml"},
			path:    ".github/workflows/ci.yml",
			want:    true,
		},
		{
			name:    "exclude-only keeps non-excluded files",
			exclude: []string{"justfile"},
			path:    "LICENSE",
			want:    true,
		},
		{
			name:    "exclude-only drops excluded file",
			exclude: []string{"justfile"},
			path:    "justfile",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filter, err := NewFileFilter(tt.include, tt.exclude)
			if err != nil {
				t.Fatalf("NewFileFilter returned error: %v", err)
			}
			if got := filter.Matches(tt.path); got != tt.want {
				t.Fatalf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileFilterMatchesManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		include      []string
		exclude      []string
		manifestPath string
		want         bool
	}{
		{
			name:         "unfiltered includes manifest",
			manifestPath: ".managedfiles",
			want:         true,
		},
		{
			name:         "unrelated include skips manifest",
			include:      []string{".github/workflows/*.yml"},
			manifestPath: ".managedfiles",
			want:         false,
		},
		{
			name:         "explicit manifest include includes manifest",
			include:      []string{".github/workflows/*.yml", ".managedfiles"},
			manifestPath: ".managedfiles",
			want:         true,
		},
		{
			name:         "wildcard matching manifest includes manifest",
			include:      []string{"*"},
			manifestPath: ".managedfiles",
			want:         true,
		},
		{
			name:         "exclude overrides explicit manifest include",
			include:      []string{"*"},
			exclude:      []string{".managedfiles"},
			manifestPath: ".managedfiles",
			want:         false,
		},
		{
			name:         "exclude-only filter skips manifest",
			exclude:      []string{"justfile"},
			manifestPath: ".managedfiles",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filter, err := NewFileFilter(tt.include, tt.exclude)
			if err != nil {
				t.Fatalf("NewFileFilter returned error: %v", err)
			}
			if got := filter.MatchesManifest(tt.manifestPath); got != tt.want {
				t.Fatalf("MatchesManifest(%q) = %v, want %v", tt.manifestPath, got, tt.want)
			}
		})
	}
}

func TestComputeFindingsWithFilter(t *testing.T) {
	configRepo := t.TempDir()
	writeTestFile(t, configRepo, "templates/LICENSE", "MIT\n")
	writeTestFile(t, configRepo, "templates/justfile", "default:\n\t@just --list\n")
	writeTestFile(t, configRepo, "templates/ci.yml", "name: ci\n")
	writeTestFile(t, configRepo, "templates/release.yml", "name: release\n")

	centralCfg := &config.CentralConfig{
		Files: []config.FileRule{
			{
				Path: ".github/workflows/ci.yml",
				Templates: []config.TemplateRef{{
					Template:     "ci.yml",
					ResolvedPath: filepath.Join(configRepo, "templates", "ci.yml"),
				}},
			},
			{
				Path: ".github/workflows/release.yml",
				Templates: []config.TemplateRef{{
					Template:     "release.yml",
					ResolvedPath: filepath.Join(configRepo, "templates", "release.yml"),
				}},
			},
			{
				Path: "LICENSE",
				Templates: []config.TemplateRef{{
					Template:     "LICENSE",
					ResolvedPath: filepath.Join(configRepo, "templates", "LICENSE"),
				}},
			},
			{
				Path: "justfile",
				Templates: []config.TemplateRef{{
					Template:     "justfile",
					ResolvedPath: filepath.Join(configRepo, "templates", "justfile"),
				}},
			},
		},
	}

	repoCfg := &config.RepoConfig{
		Name:          "demo",
		DefaultBranch: "main",
	}

	t.Run("include subset excludes unrelated rules and manifest", func(t *testing.T) {
		filter, err := NewFileFilter([]string{".github/workflows/*.yml"}, []string{".github/workflows/release.yml"})
		if err != nil {
			t.Fatalf("NewFileFilter error: %v", err)
		}

		findings, err := ComputeFindingsWithFilter(repoCfg, centralCfg, t.TempDir(), config.ManagedFilesManifestName, filter)
		if err != nil {
			t.Fatalf("ComputeFindingsWithFilter error: %v", err)
		}
		if len(findings) != 1 || findings[0].FilePath != ".github/workflows/ci.yml" {
			t.Fatalf("findings = %+v, want only .github/workflows/ci.yml", findings)
		}
	})

	t.Run("include explicitly matching manifest includes manifest", func(t *testing.T) {
		filter, err := NewFileFilter([]string{"LICENSE", ".managedfiles"}, nil)
		if err != nil {
			t.Fatalf("NewFileFilter error: %v", err)
		}

		findings, err := ComputeFindingsWithFilter(repoCfg, centralCfg, t.TempDir(), config.ManagedFilesManifestName, filter)
		if err != nil {
			t.Fatalf("ComputeFindingsWithFilter error: %v", err)
		}
		if len(findings) != 2 {
			t.Fatalf("len(findings) = %d, want 2 (%+v)", len(findings), findings)
		}
		if findings[0].FilePath != "LICENSE" || findings[1].FilePath != ".managedfiles" {
			t.Fatalf("findings = %+v, want LICENSE and .managedfiles", findings)
		}
	})

	t.Run("exclude-only skips excluded files and manifest", func(t *testing.T) {
		filter, err := NewFileFilter(nil, []string{"justfile", ".github/**"})
		if err != nil {
			t.Fatalf("NewFileFilter error: %v", err)
		}

		findings, err := ComputeFindingsWithFilter(repoCfg, centralCfg, t.TempDir(), config.ManagedFilesManifestName, filter)
		if err != nil {
			t.Fatalf("ComputeFindingsWithFilter error: %v", err)
		}
		if len(findings) != 1 || findings[0].FilePath != "LICENSE" {
			t.Fatalf("findings = %+v, want only LICENSE", findings)
		}
	})
}
