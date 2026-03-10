package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/discovery"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/engine"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/gitops"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/schema"
)

func runApply(version string, args []string) {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	repoFlag := fs.String("repo", "", "Target a single repo by folder name")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	workspaceDir, err := os.Getwd()
	if err != nil {
		output.Error(fmt.Sprintf("failed to get working directory: %v", err))
		os.Exit(1)
	}

	rootCfg, err := config.LoadRootConfig(workspaceDir)
	if err != nil {
		output.Error(fmt.Sprintf("root config error: %v", err))
		os.Exit(1)
	}

	configRepoPath := rootCfg.ResolveConfigRepoPath(workspaceDir)
	centralCfg, err := config.LoadCentralConfig(configRepoPath)
	if err != nil {
		output.Error(fmt.Sprintf("central config error: %v", err))
		os.Exit(1)
	}

	report := output.NewReport(version, "apply", filepath.Join(workspaceDir, config.RootConfigFileName), configRepoPath)

	var repos []string
	if *repoFlag != "" {
		repoPath := filepath.Join(workspaceDir, *repoFlag)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			output.Error(fmt.Sprintf("repo %q not found", *repoFlag))
			os.Exit(1)
		}
		repos = []string{repoPath}
	} else {
		repos, err = discovery.DiscoverRepos(workspaceDir, rootCfg.Excludes)
		if err != nil {
			output.Error(fmt.Sprintf("repo discovery error: %v", err))
			os.Exit(1)
		}
	}

	if !*jsonFlag {
		output.Header("gitrepoforge apply")
		fmt.Println()
	}

	for _, repoPath := range repos {
		repoName := filepath.Base(repoPath)
		result := applyRepo(repoPath, repoName, rootCfg, centralCfg, configRepoPath)
		report.Repos = append(report.Repos, result)
	}

	if *jsonFlag {
		report.PrintJSON()
	} else {
		report.PrintHuman(false)
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}

func applyRepo(repoPath, repoName string, rootCfg *config.RootConfig, centralCfg *config.CentralConfig, configRepoPath string) output.RepoResult {
	if !config.RepoConfigExists(repoPath) {
		return output.RepoResult{
			Name:   repoName,
			Status: "skipped",
		}
	}

	repoCfg, err := config.LoadRepoConfig(repoPath)
	if err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "invalid",
			ValidationErrors: []string{err.Error()},
		}
	}

	validationErrors := schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath)
	if len(validationErrors) > 0 {
		var errStrs []string
		for _, e := range validationErrors {
			errStrs = append(errStrs, e.Error())
		}
		return output.RepoResult{
			Name:             repoName,
			Status:           "invalid",
			ValidationErrors: errStrs,
		}
	}

	// Check repo is clean
	clean, err := gitops.IsClean(repoPath)
	if err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to check repo status: %v", err)},
		}
	}
	if !clean {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{"repo has uncommitted changes; apply requires a clean working tree"},
		}
	}

	findings, err := engine.ComputeFindings(repoCfg, centralCfg, repoPath)
	if err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{err.Error()},
		}
	}

	if len(findings) == 0 {
		return output.RepoResult{
			Name:   repoName,
			Status: "clean",
		}
	}

	// Create branch, apply changes, commit, push
	branchName := rootCfg.Git.BranchPrefix + "update"

	// Checkout default branch first
	if err := gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch); err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to checkout default branch: %v", err)},
		}
	}

	// Check if remote branch already exists
	remoteBranchExists, err := gitops.RemoteBranchExists(repoPath, branchName)
	if err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to check remote branch: %v", err)},
		}
	}

	// Create branch
	if err := gitops.CreateBranch(repoPath, branchName); err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to create branch: %v", err)},
		}
	}

	// Apply changes
	if err := engine.ApplyFindings(findings, repoPath); err != nil {
		// Attempt to return to default branch
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to apply changes: %v", err)},
		}
	}

	// Stage and commit
	if err := gitops.AddAll(repoPath); err != nil {
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to stage changes: %v", err)},
		}
	}

	hasChanges, err := gitops.HasChanges(repoPath)
	if err != nil {
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to check for changes: %v", err)},
		}
	}

	if !hasChanges {
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:   repoName,
			Status: "clean",
		}
	}

	if err := gitops.Commit(repoPath, "gitrepoforge: apply desired state"); err != nil {
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to commit: %v", err)},
		}
	}

	// Push
	if err := gitops.Push(repoPath, branchName); err != nil {
		gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to push: %v", err)},
		}
	}

	// Create PR if configured and remote branch didn't already exist
	if rootCfg.Git.PullRequest == config.PullRequestGitHubCLI {
		if remoteBranchExists {
			output.Warning(fmt.Sprintf("%s: remote branch %s already exists; skipping PR creation", repoName, branchName))
		} else {
			err := gitops.CreatePR(repoPath, branchName, repoCfg.DefaultBranch,
				rootCfg.Git.PRTitle,
				rootCfg.Git.PRBody)
			if err != nil {
				output.Warning(fmt.Sprintf("%s: PR creation failed: %v", repoName, err))
			}
		}
	}

	// Return to default branch
	gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch)

	var findingOutputs []output.FindingOutput
	for _, f := range findings {
		findingOutputs = append(findingOutputs, output.FindingOutput{
			FilePath:  f.FilePath,
			Operation: f.Operation,
			Message:   f.Message,
		})
	}

	return output.RepoResult{
		Name:     repoName,
		Status:   "applied",
		Findings: findingOutputs,
	}
}
