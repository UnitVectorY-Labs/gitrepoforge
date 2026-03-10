package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/engine"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/gitops"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/schema"
)

func runBootstrap(version string, args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	repoFlag := fs.String("repo", "", "Target repo by folder name (required)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	if *repoFlag == "" {
		output.Error("bootstrap requires --repo flag")
		os.Exit(1)
	}

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

	repoPath := filepath.Join(workspaceDir, *repoFlag)
	report := output.NewReport(version, "bootstrap", filepath.Join(workspaceDir, config.RootConfigFileName), configRepoPath)

	if !*jsonFlag {
		output.Header("gitrepoforge bootstrap")
		fmt.Println()
	}

	result := bootstrapRepo(repoPath, *repoFlag, rootCfg, centralCfg, configRepoPath)
	report.Repos = append(report.Repos, result)

	if *jsonFlag {
		report.PrintJSON()
	} else {
		report.PrintHuman(false)
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}

func bootstrapRepo(repoPath, repoName string, rootCfg *config.RootConfig, centralCfg *config.CentralConfig, configRepoPath string) output.RepoResult {
	if !config.RepoConfigExists(repoPath) {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{"no .gitrepoforge file found; bootstrap requires a repo config"},
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
			ValidationErrors: []string{"repo has uncommitted changes; bootstrap requires a clean working tree"},
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

	// Bootstrap uses a distinct branch name
	branchName := rootCfg.Git.BranchPrefix + "bootstrap"

	// Checkout default branch first
	if err := gitops.CheckoutBranch(repoPath, repoCfg.DefaultBranch); err != nil {
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to checkout default branch: %v", err)},
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

	if err := gitops.Commit(repoPath, "gitrepoforge: bootstrap repo"); err != nil {
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

	// Create PR if configured
	if rootCfg.Git.PullRequest == config.PullRequestGitHubCLI {
		err := gitops.CreatePR(repoPath, branchName, repoCfg.DefaultBranch,
			rootCfg.Git.BootstrapPRTitle,
			rootCfg.Git.BootstrapPRBody)
		if err != nil {
			output.Warning(fmt.Sprintf("%s: PR creation failed: %v", repoName, err))
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
