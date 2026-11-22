package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func TestAddPathsStagesCreatesUpdatesAndDeletesOnlyForTargetPaths(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(dir, "update.txt"), []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "delete.txt"), []byte("remove me\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untouched.txt"), []byte("v1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify multiple files, including one not passed to AddPaths.
	if err := os.WriteFile(filepath.Join(dir, "create.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "update.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untouched.txt"), []byte("v2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := AddPaths(dir, []string{"create.txt", "update.txt", "delete.txt", "update.txt"}); err != nil {
		t.Fatalf("AddPaths returned error: %v", err)
	}

	staged := runGit(t, dir, "diff", "--cached", "--name-only")
	for _, want := range []string{"create.txt", "update.txt", "delete.txt"} {
		if !strings.Contains(staged, want) {
			t.Fatalf("expected %q in staged files, got:\n%s", want, staged)
		}
	}
	if strings.Contains(staged, "untouched.txt") {
		t.Fatalf("did not expect untouched.txt to be staged, got:\n%s", staged)
	}
}
