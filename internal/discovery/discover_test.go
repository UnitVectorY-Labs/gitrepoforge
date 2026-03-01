package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to create a fake git repo (directory with .git subdirectory)
func createFakeRepo(t *testing.T, base, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(base, name, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// helper to create a plain directory (no .git)
func createPlainDir(t *testing.T, base, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(base, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverRepos(t *testing.T) {
	tests := []struct {
		name     string
		repos    []string // dirs that get a .git subdir
		dirs     []string // dirs without .git
		excludes []string
		want     []string // expected repo base names, sorted
	}{
		{
			name:  "multiple repos and non-repos",
			repos: []string{"alpha", "charlie", "bravo"},
			dirs:  []string{"not-a-repo", "also-plain"},
			want:  []string{"alpha", "bravo", "charlie"},
		},
		{
			name:  "dot-named repos are included",
			repos: []string{".github", "regular"},
			want:  []string{".github", "regular"},
		},
		{
			name:     "exclude exact name",
			repos:    []string{"keep", "remove"},
			excludes: []string{"remove"},
			want:     []string{"keep"},
		},
		{
			name:     "exclude glob pattern",
			repos:    []string{"test-one", "test-two", "prod"},
			excludes: []string{"test-*"},
			want:     []string{"prod"},
		},
		{
			name:     "exclude multiple patterns",
			repos:    []string{"foo", "bar", "test-a", "dev-b"},
			excludes: []string{"test-*", "dev-*"},
			want:     []string{"bar", "foo"},
		},
		{
			name: "empty workspace",
			want: []string{},
		},
		{
			name: "only non-repo directories",
			dirs: []string{"dir1", "dir2"},
			want: []string{},
		},
		{
			name:     "exclude all repos",
			repos:    []string{"a", "b"},
			excludes: []string{"a", "b"},
			want:     []string{},
		},
		{
			name:  "results are sorted alphabetically",
			repos: []string{"zulu", "mike", "alpha", "delta"},
			want:  []string{"alpha", "delta", "mike", "zulu"},
		},
		{
			name:     "case-insensitive exclude match",
			repos:    []string{"MyRepo", "other"},
			excludes: []string{"myrepo"},
			want:     []string{"other"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			for _, r := range tc.repos {
				createFakeRepo(t, workspace, r)
			}
			for _, d := range tc.dirs {
				createPlainDir(t, workspace, d)
			}

			got, err := DiscoverRepos(workspace, tc.excludes)
			if err != nil {
				t.Fatalf("DiscoverRepos() error: %v", err)
			}

			// build expected full paths
			want := make([]string, len(tc.want))
			for i, name := range tc.want {
				want[i] = filepath.Join(workspace, name)
			}

			if len(got) != len(want) {
				t.Fatalf("got %d repos %v, want %d repos %v", len(got), got, len(want), want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
				}
			}
		})
	}
}

func TestDiscoverRepos_InvalidWorkspace(t *testing.T) {
	_, err := DiscoverRepos("/nonexistent/path/xyz", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent workspace, got nil")
	}
}

func TestIsGitRepo(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, base string)
		want    bool
	}{
		{
			name: "directory with .git subdir",
			setup: func(t *testing.T, base string) {
				os.MkdirAll(filepath.Join(base, ".git"), 0o755)
			},
			want: true,
		},
		{
			name: "directory without .git",
			setup: func(t *testing.T, base string) {
				// empty dir, nothing to do
			},
			want: false,
		},
		{
			name: ".git is a file not a directory",
			setup: func(t *testing.T, base string) {
				os.WriteFile(filepath.Join(base, ".git"), []byte("gitdir: ../other"), 0o644)
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			if got := isGitRepo(dir); got != tc.want {
				t.Errorf("isGitRepo() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		excludes []string
		want     bool
	}{
		{
			name:     "no excludes",
			input:    "repo",
			excludes: nil,
			want:     false,
		},
		{
			name:     "exact match",
			input:    "secret",
			excludes: []string{"secret"},
			want:     true,
		},
		{
			name:     "glob star match",
			input:    "test-integration",
			excludes: []string{"test-*"},
			want:     true,
		},
		{
			name:     "glob question mark",
			input:    "ab",
			excludes: []string{"a?"},
			want:     true,
		},
		{
			name:     "no match",
			input:    "production",
			excludes: []string{"test-*", "dev-*"},
			want:     false,
		},
		{
			name:     "case-insensitive fallback",
			input:    "MyRepo",
			excludes: []string{"myrepo"},
			want:     true,
		},
		{
			name:     "multiple patterns first matches",
			input:    "dev-tools",
			excludes: []string{"dev-*", "test-*"},
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			input:    "test-unit",
			excludes: []string{"dev-*", "test-*"},
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExcluded(tc.input, tc.excludes); got != tc.want {
				t.Errorf("isExcluded(%q, %v) = %v, want %v", tc.input, tc.excludes, got, tc.want)
			}
		})
	}
}
