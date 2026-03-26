package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/engine"
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
	report.IgnoreMissing = rootCfg.IgnoreMissing

	if !*jsonFlag {
		output.Header("gitrepoforge bootstrap")
		fmt.Println()
	}

	result := bootstrapRepo(repoPath, *repoFlag, rootCfg, centralCfg)
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

func bootstrapRepo(repoPath, repoName string, rootCfg *config.RootConfig, centralCfg *config.CentralConfig) output.RepoResult {
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

	if gitValidationErrors := validateRootGitTemplates(rootCfg, repoCfg); len(gitValidationErrors) > 0 {
		return output.RepoResult{
			Name:             repoName,
			Status:           "invalid",
			ValidationErrors: gitValidationErrors,
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
		result := output.RepoResult{
			Name:   repoName,
			Status: "clean",
		}
		if !rootCfg.Git.Commit {
			result.StatusDetail = cleanStatusDetail(repoPath)
		}
		return result
	}

	return applyFindingsWithGit(repoPath, repoName, repoCfg, rootCfg, findings)
}
