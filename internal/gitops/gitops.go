package gitops

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsClean checks if the repo working tree is clean (no uncommitted changes).
func IsClean(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(out)) == "", nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CheckoutBranch checks out a branch.
func CheckoutBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s failed: %s: %w", branch, string(out), err)
	}
	return nil
}

// CreateBranch creates and checks out a new branch from the current HEAD.
func CreateBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout -b %s failed: %s: %w", branch, string(out), err)
	}
	return nil
}

// AddAll stages all changes.
func AddAll(repoPath string) error {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s: %w", string(out), err)
	}
	return nil
}

// Commit creates a commit with the given message.
func Commit(repoPath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s: %w", string(out), err)
	}
	return nil
}

// Push pushes the current branch to the specified remote.
func Push(repoPath, remote, branch string) error {
	cmd := exec.Command("git", "push", remote, branch)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s: %w", string(out), err)
	}
	return nil
}

// DeleteBranch deletes a local branch.
func DeleteBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -D %s failed: %s: %w", branch, string(out), err)
	}
	return nil
}

// HasChanges checks if there are staged changes to commit.
func HasChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return true, nil
			}
		}
		return false, fmt.Errorf("git diff --cached failed: %w", err)
	}
	return false, nil
}

// CreatePR uses gh CLI to create a pull request from the current branch.
func CreatePR(repoPath string) error {
	cmd := exec.Command("gh", "pr", "create", "--fill")
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr create failed: %s: %w", string(out), err)
	}
	return nil
}
