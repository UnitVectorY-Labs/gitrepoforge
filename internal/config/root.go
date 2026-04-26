package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitConfig controls the Git automation performed during apply for a named action.
type GitConfig struct {
	CreateBranch           bool   `yaml:"create_branch"`
	BranchName             string `yaml:"branch_name"`
	Commit                 bool   `yaml:"commit"`
	CommitMessage          string `yaml:"commit_message"`
	OnDefaultBranch        bool   `yaml:"on_default_branch"`
	Push                   bool   `yaml:"push"`
	Remote                 string `yaml:"remote"`
	PullRequest            string `yaml:"pull_request"`
	ReturnToOriginalBranch bool   `yaml:"return_to_original_branch"`
	DeleteBranch           bool   `yaml:"delete_branch"`
}

// ReportConfig controls the behavior of the report command.
type ReportConfig struct {
	CollapseDiffs bool `yaml:"collapse_diffs"`
}

// RootConfig represents the root config dotfile (.gitrepoforge-config)
// that lives in the checkout root (workspace directory).
type RootConfig struct {
	ConfigRepo    string               `yaml:"config_repo"`
	Excludes      []string             `yaml:"excludes"`
	IgnoreMissing bool                 `yaml:"ignore_missing"`
	Manifest      string               `yaml:"manifest"`
	Actions       map[string]GitConfig `yaml:"-"`
	Report        ReportConfig         `yaml:"-"`
}

const RootConfigFileName = ".gitrepoforge-config"

type rawReportConfig struct {
	CollapseDiffs *bool `yaml:"collapse_diffs"`
}

type rawRootConfig struct {
	ConfigRepo    string               `yaml:"config_repo"`
	Excludes      []string             `yaml:"excludes"`
	IgnoreMissing bool                 `yaml:"ignore_missing"`
	Manifest      string               `yaml:"manifest"`
	Report        rawReportConfig      `yaml:"report"`
	Action        map[string]GitConfig `yaml:"action"`
}

func LoadRootConfig(workspaceDir string) (*RootConfig, error) {
	path := filepath.Join(workspaceDir, RootConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read root config %s: %w", path, err)
	}

	var raw rawRootConfig
	if err := unmarshalYAMLKnownFields(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse root config %s: %w", path, err)
	}
	cfg := RootConfig{
		ConfigRepo:    raw.ConfigRepo,
		Excludes:      raw.Excludes,
		IgnoreMissing: raw.IgnoreMissing,
		Manifest:      raw.Manifest,
	}
	cfg.Report = ReportConfig{
		CollapseDiffs: true, // default
	}
	if raw.Report.CollapseDiffs != nil {
		cfg.Report.CollapseDiffs = *raw.Report.CollapseDiffs
	}
	if cfg.ConfigRepo == "" {
		return nil, fmt.Errorf("root config %s: config_repo is required", path)
	}
	cfg.Manifest, err = validateAndNormalizeManifestPath(cfg.Manifest)
	if err != nil {
		return nil, fmt.Errorf("root config %s: %w", path, err)
	}
	actions := make(map[string]GitConfig, len(raw.Action))
	for name, gitCfg := range raw.Action {
		if err := validateGitConfig(&gitCfg); err != nil {
			return nil, fmt.Errorf("root config %s: action.%s: %w", path, name, err)
		}
		actions[name] = gitCfg
	}
	cfg.Actions = actions

	return &cfg, nil
}

// ResolveAction returns the GitConfig for the named action. If name is empty,
// a zero GitConfig (no git automation) is returned. An error is returned when
// name is non-empty but does not match any configured action.
func (rc *RootConfig) ResolveAction(name string) (*GitConfig, error) {
	if name == "" {
		return &GitConfig{}, nil
	}
	action, ok := rc.Actions[name]
	if !ok {
		return nil, fmt.Errorf("action %q is not defined in the action config", name)
	}
	return &action, nil
}

func (g *GitConfig) Normalize() {
	if strings.TrimSpace(g.PullRequest) == "" {
		g.PullRequest = PullRequestNo
		return
	}
	g.PullRequest = strings.ToUpper(strings.TrimSpace(g.PullRequest))
}

func (g *GitConfig) GitOptionsSpecified() bool {
	return g.CreateBranch ||
		g.Commit ||
		g.Push ||
		g.ReturnToOriginalBranch ||
		g.DeleteBranch ||
		(strings.TrimSpace(g.PullRequest) != "" && strings.ToUpper(strings.TrimSpace(g.PullRequest)) != PullRequestNo)
}

func (g *GitConfig) BuildBranchName(values map[string]string) string {
	return substituteGitPlaceholders(g.BranchName, values)
}

func (g *GitConfig) BuildCommitMessage(values map[string]string) string {
	return substituteGitPlaceholders(g.CommitMessage, values)
}

func validateGitConfig(g *GitConfig) error {
	g.Normalize()

	if g.PullRequest != PullRequestNo && g.PullRequest != PullRequestGitHubCLI {
		return fmt.Errorf("git.pull_request must be %q or %q", PullRequestNo, PullRequestGitHubCLI)
	}
	if g.CreateBranch && strings.TrimSpace(g.BranchName) == "" {
		return fmt.Errorf("git.branch_name is required when git.create_branch is true")
	}
	if g.Commit && strings.TrimSpace(g.CommitMessage) == "" {
		return fmt.Errorf("git.commit_message is required when git.commit is true")
	}
	if g.Push && strings.TrimSpace(g.Remote) == "" {
		return fmt.Errorf("git.remote is required when git.push is true")
	}
	if g.PullRequest == PullRequestGitHubCLI && !g.Push {
		return fmt.Errorf("git.pull_request requires git.push to be true")
	}
	if g.ReturnToOriginalBranch && !g.CreateBranch {
		return fmt.Errorf("git.return_to_original_branch requires git.create_branch to be true")
	}
	if g.DeleteBranch && !g.ReturnToOriginalBranch {
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
