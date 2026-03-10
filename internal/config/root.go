package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GitConfig controls the Git automation performed during apply and bootstrap
// commands. It mirrors the git configuration section from the repver tool.
type GitConfig struct {
	BranchPrefix           string `yaml:"branch_prefix"`
	CommitMessage          string `yaml:"commit_message"`
	BootstrapCommitMessage string `yaml:"bootstrap_commit_message"`
	Push                   *bool  `yaml:"push"`
	Remote                 string `yaml:"remote"`
	PullRequest            string `yaml:"pull_request"`
	PRTitle                string `yaml:"pr_title"`
	PRBody                 string `yaml:"pr_body"`
	BootstrapPRTitle       string `yaml:"bootstrap_pr_title"`
	BootstrapPRBody        string `yaml:"bootstrap_pr_body"`
	ReturnToOriginalBranch *bool  `yaml:"return_to_original_branch"`
	DeleteBranch           bool   `yaml:"delete_branch"`
}

// RootConfig represents the root config dotfile (.gitrepoforge-config)
// that lives in the checkout root (workspace directory).
type RootConfig struct {
	ConfigRepo string    `yaml:"config_repo"`
	Excludes   []string  `yaml:"excludes"`
	Git        GitConfig `yaml:"git"`
}

const RootConfigFileName = ".gitrepoforge-config"

// rawRootConfig is used internally to handle backward compatibility with the
// legacy top-level branch_prefix and create_pr fields.
type rawRootConfig struct {
	ConfigRepo string    `yaml:"config_repo"`
	Excludes   []string  `yaml:"excludes"`
	Git        GitConfig `yaml:"git"`

	// Deprecated: use Git.BranchPrefix
	BranchPrefix string `yaml:"branch_prefix"`
	// Deprecated: use Git.PullRequest
	CreatePR *bool `yaml:"create_pr"`
}

func LoadRootConfig(workspaceDir string) (*RootConfig, error) {
	path := filepath.Join(workspaceDir, RootConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read root config %s: %w", path, err)
	}
	var raw rawRootConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse root config %s: %w", path, err)
	}
	if raw.ConfigRepo == "" {
		return nil, fmt.Errorf("root config %s: config_repo is required", path)
	}

	cfg := &RootConfig{
		ConfigRepo: raw.ConfigRepo,
		Excludes:   raw.Excludes,
		Git:        raw.Git,
	}

	// Migrate deprecated top-level fields into the git section
	if cfg.Git.BranchPrefix == "" && raw.BranchPrefix != "" {
		cfg.Git.BranchPrefix = raw.BranchPrefix
	}
	if cfg.Git.PullRequest == "" && raw.CreatePR != nil {
		if *raw.CreatePR {
			cfg.Git.PullRequest = PullRequestGitHubCLI
		} else {
			cfg.Git.PullRequest = PullRequestNo
		}
	}

	applyGitDefaults(&cfg.Git)

	if err := validateGitConfig(&cfg.Git); err != nil {
		return nil, fmt.Errorf("root config %s: %w", path, err)
	}

	return cfg, nil
}

const (
	// PullRequestNo disables pull request creation.
	PullRequestNo = "NO"
	// PullRequestGitHubCLI creates pull requests using the GitHub CLI.
	PullRequestGitHubCLI = "GITHUB_CLI"
)

func applyGitDefaults(g *GitConfig) {
	if g.BranchPrefix == "" {
		g.BranchPrefix = "gitrepoforge/"
	}
	if g.CommitMessage == "" {
		g.CommitMessage = "gitrepoforge: apply desired state"
	}
	if g.BootstrapCommitMessage == "" {
		g.BootstrapCommitMessage = "gitrepoforge: bootstrap repo"
	}
	if g.Push == nil {
		t := true
		g.Push = &t
	}
	if g.Remote == "" {
		g.Remote = "origin"
	}
	if g.PullRequest == "" {
		g.PullRequest = PullRequestNo
	}
	if g.PRTitle == "" {
		g.PRTitle = g.CommitMessage
	}
	if g.PRBody == "" {
		g.PRBody = "Automated changes applied by gitrepoforge."
	}
	if g.BootstrapPRTitle == "" {
		g.BootstrapPRTitle = g.BootstrapCommitMessage
	}
	if g.BootstrapPRBody == "" {
		g.BootstrapPRBody = "Automated bootstrap by gitrepoforge."
	}
	if g.ReturnToOriginalBranch == nil {
		t := true
		g.ReturnToOriginalBranch = &t
	}
}

func validateGitConfig(g *GitConfig) error {
	pr := strings.ToUpper(g.PullRequest)
	if pr != PullRequestNo && pr != PullRequestGitHubCLI {
		return fmt.Errorf("git.pull_request must be %q or %q, got %q", PullRequestNo, PullRequestGitHubCLI, g.PullRequest)
	}
	g.PullRequest = pr

	if !*g.Push && g.PullRequest == PullRequestGitHubCLI {
		return fmt.Errorf("git.pull_request cannot be %q when git.push is false", PullRequestGitHubCLI)
	}
	if g.DeleteBranch && !*g.ReturnToOriginalBranch {
		return fmt.Errorf("git.delete_branch requires git.return_to_original_branch to be true")
	}
	return nil
}

// ResolveConfigRepoPath resolves the config repo path (relative to workspace or absolute).
func (rc *RootConfig) ResolveConfigRepoPath(workspaceDir string) string {
	if filepath.IsAbs(rc.ConfigRepo) {
		return rc.ConfigRepo
	}
	return filepath.Join(workspaceDir, rc.ConfigRepo)
}
