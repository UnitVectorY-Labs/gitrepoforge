package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/discovery"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/engine"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/schema"
)

func runValidate(version string, args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	repoFlag := fs.String("repo", "", "Target a single repo by folder name")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")
	verboseFlag := fs.Bool("verbose", false, "Print colorized line diffs for drift findings")
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

	report := output.NewReport(version, "validate", filepath.Join(workspaceDir, config.RootConfigFileName), configRepoPath)
	report.IgnoreMissing = rootCfg.IgnoreMissing

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
		output.Header("gitrepoforge validate")
		fmt.Println()
	}

	for _, repoPath := range repos {
		repoName := filepath.Base(repoPath)

		if !config.RepoConfigExists(repoPath) {
			report.Repos = append(report.Repos, output.RepoResult{
				Name:   repoName,
				Status: "skipped",
			})
			continue
		}

		repoCfg, err := config.LoadRepoConfig(repoPath)
		if err != nil {
			report.Repos = append(report.Repos, output.RepoResult{
				Name:             repoName,
				Status:           "invalid",
				ValidationErrors: []string{err.Error()},
			})
			continue
		}

		validationErrors := schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath)
		if len(validationErrors) > 0 {
			var errStrs []string
			for _, e := range validationErrors {
				errStrs = append(errStrs, e.Error())
			}
			report.Repos = append(report.Repos, output.RepoResult{
				Name:             repoName,
				Status:           "invalid",
				ValidationErrors: errStrs,
			})
			continue
		}

		findings, err := engine.ComputeFindings(repoCfg, centralCfg, repoPath, config.ResolveManifestPath(rootCfg, repoCfg))
		if err != nil {
			report.Repos = append(report.Repos, output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{err.Error()},
			})
			continue
		}

		if len(findings) == 0 {
			result := output.RepoResult{
				Name:         repoName,
				Status:       "clean",
				StatusDetail: cleanStatusDetail(repoPath),
			}
			report.Repos = append(report.Repos, result)
		} else {
			var findingOutputs []output.FindingOutput
			for _, f := range findings {
				findingOutputs = append(findingOutputs, output.FindingOutput{
					FilePath:  f.FilePath,
					Operation: f.Operation,
					Message:   f.Message,
					Expected:  f.Expected,
					Actual:    f.Actual,
				})
			}
			report.Repos = append(report.Repos, output.RepoResult{
				Name:     repoName,
				Status:   "drift",
				Findings: findingOutputs,
			})
		}
	}

	if *jsonFlag {
		report.PrintJSON()
	} else {
		report.PrintHuman(*verboseFlag)
	}

	if report.HasFailures() {
		os.Exit(1)
	}
}
