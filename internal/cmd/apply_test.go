package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
)

func writeCmdTestFile(t *testing.T, dir, relPath, content string) {
	t.Helper()

	fullPath := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create parent directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

func loadApplyTestCentralConfig(t *testing.T) *config.CentralConfig {
	t.Helper()

	dir := t.TempDir()
	writeCmdTestFile(t, dir, "outputs/README.md.gitrepoforge", `templates:
  - template: README.md.tmpl
`)
	writeCmdTestFile(t, dir, "templates/README.md.tmpl", "managed readme\n")

	centralCfg, err := config.LoadCentralConfig(dir)
	if err != nil {
		t.Fatalf("LoadCentralConfig returned error: %v", err)
	}
	return centralCfg
}

func createApplyTestRepo(t *testing.T) string {
	t.Helper()

	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "demo")
	writeCmdTestFile(t, repoDir, config.RepoConfigFileName, `name: demo
default_branch: main
config: {}
`)
	return repoDir
}

func resultHasFindingPath(result output.RepoResult, path string) bool {
	for _, finding := range result.Findings {
		if finding.FilePath == path {
			return true
		}
	}
	return false
}

func runApplyTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func initApplyTestGitRepo(t *testing.T, repoDir string) {
	t.Helper()

	runApplyTestGit(t, repoDir, "init", "-b", "main")
	runApplyTestGit(t, repoDir, "config", "user.name", "Test User")
	runApplyTestGit(t, repoDir, "config", "user.email", "test@example.com")
	runApplyTestGit(t, repoDir, "add", "-A")
	runApplyTestGit(t, repoDir, "commit", "-m", "initial commit")
}

func TestResolveApplyActionDefaultsToDryRunWhenFlagOmitted(t *testing.T) {
	rootCfg := &config.RootConfig{}

	gitCfg, actionName, err := resolveApplyAction(rootCfg, "", false)
	if err != nil {
		t.Fatalf("resolveApplyAction returned error: %v", err)
	}
	if actionName != "" {
		t.Fatalf("actionName = %q, want empty", actionName)
	}
	if gitCfg == nil {
		t.Fatal("gitCfg = nil, want non-nil")
	}
	if gitCfg.GitOptionsSpecified() {
		t.Fatal("expected no git automation when --action is omitted")
	}
}

func TestResolveApplyActionRejectsEmptySpecifiedAction(t *testing.T) {
	rootCfg := &config.RootConfig{
		Actions: map[string]config.GitConfig{
			"stage": {},
		},
	}

	_, _, err := resolveApplyAction(rootCfg, "", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "available actions: stage") {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestApplyRepoWithoutActionReportsDriftWithoutWriting(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)

	result := applyRepo(repoDir, filepath.Base(repoDir), &config.RootConfig{}, &config.GitConfig{}, "", centralCfg)
	if result.Status != "drift" {
		t.Fatalf("Status = %q, want %q", result.Status, "drift")
	}
	if len(result.Findings) != 2 {
		t.Fatalf("Findings length = %d, want 2", len(result.Findings))
	}
	if !resultHasFindingPath(result, "README.md") {
		t.Fatal("expected README.md finding")
	}
	if !resultHasFindingPath(result, config.ManagedFilesManifestName) {
		t.Fatalf("expected %s finding", config.ManagedFilesManifestName)
	}

	readmePath := filepath.Join(repoDir, "README.md")
	if _, err := os.Stat(readmePath); !os.IsNotExist(err) {
		t.Fatalf("README.md should not have been written, stat err = %v", err)
	}

	manifestPath := filepath.Join(repoDir, config.ManagedFilesManifestName)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatalf("%s should not have been written, stat err = %v", config.ManagedFilesManifestName, err)
	}
}

func TestApplyRepoWithNamedActionAppliesChanges(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)

	result := applyRepo(repoDir, filepath.Base(repoDir), &config.RootConfig{}, &config.GitConfig{}, "stage", centralCfg)
	if result.Status != "applied" {
		t.Fatalf("Status = %q, want %q", result.Status, "applied")
	}

	readmePath := filepath.Join(repoDir, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	if string(content) != "managed readme\n" {
		t.Fatalf("README.md = %q, want %q", string(content), "managed readme\n")
	}

	manifestPath := filepath.Join(repoDir, config.ManagedFilesManifestName)
	manifestContent, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", config.ManagedFilesManifestName, err)
	}
	manifestText := string(manifestContent)
	if !strings.Contains(manifestText, "path: "+config.ManagedFilesManifestName) {
		t.Fatalf("%s should reference itself, got %q", config.ManagedFilesManifestName, manifestText)
	}
	if !strings.Contains(manifestText, "path: README.md") {
		t.Fatalf("%s should include README.md, got %q", config.ManagedFilesManifestName, manifestText)
	}
}

func TestApplyRepoUsesWorkspaceManifestOverride(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)
	rootCfg := &config.RootConfig{Manifest: ".workspace-managedfiles"}

	result := applyRepo(repoDir, filepath.Base(repoDir), rootCfg, &config.GitConfig{}, "stage", centralCfg)
	if result.Status != "applied" {
		t.Fatalf("Status = %q, want %q", result.Status, "applied")
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".workspace-managedfiles")); err != nil {
		t.Fatalf("expected workspace manifest to be written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, config.ManagedFilesManifestName)); !os.IsNotExist(err) {
		t.Fatalf("default manifest should not have been written, stat err = %v", err)
	}
}

func TestApplyRepoUsesRepoManifestOverrideOverWorkspace(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)
	writeCmdTestFile(t, repoDir, config.RepoConfigFileName, `name: demo
default_branch: main
manifest: .repo-managedfiles
config: {}
`)

	result := applyRepo(repoDir, filepath.Base(repoDir), &config.RootConfig{Manifest: ".workspace-managedfiles"}, &config.GitConfig{}, "stage", centralCfg)
	if result.Status != "applied" {
		t.Fatalf("Status = %q, want %q", result.Status, "applied")
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".repo-managedfiles")); err != nil {
		t.Fatalf("expected repo manifest to be written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".workspace-managedfiles")); !os.IsNotExist(err) {
		t.Fatalf("workspace manifest should not have been written, stat err = %v", err)
	}
}

func TestApplyRepoWithOnDefaultBranchAppliesChangesOnDefaultBranch(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)
	initApplyTestGitRepo(t, repoDir)

	result := applyRepo(repoDir, filepath.Base(repoDir), &config.RootConfig{}, &config.GitConfig{OnDefaultBranch: true}, "stage", centralCfg)
	if result.Status != "applied" {
		t.Fatalf("Status = %q, want %q", result.Status, "applied")
	}
}

func TestApplyRepoWithOnDefaultBranchFailsOffDefaultBranch(t *testing.T) {
	centralCfg := loadApplyTestCentralConfig(t)
	repoDir := createApplyTestRepo(t)
	initApplyTestGitRepo(t, repoDir)
	runApplyTestGit(t, repoDir, "checkout", "-b", "feature/test")

	result := applyRepo(repoDir, filepath.Base(repoDir), &config.RootConfig{}, &config.GitConfig{OnDefaultBranch: true}, "stage", centralCfg)
	if result.Status != "failed" {
		t.Fatalf("Status = %q, want %q", result.Status, "failed")
	}
	if len(result.ValidationErrors) != 1 {
		t.Fatalf("ValidationErrors length = %d, want 1", len(result.ValidationErrors))
	}
	if !strings.Contains(result.ValidationErrors[0], `action requires default branch "main"`) {
		t.Fatalf("unexpected validation error %q", result.ValidationErrors[0])
	}
}
