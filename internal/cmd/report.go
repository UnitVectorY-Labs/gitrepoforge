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

func runReport(version string, args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	repoFlag := fs.String("repo", "", "Target a single repo by folder name")
	outputFlag := fs.String("output", "", "Write report to a file instead of stdout")
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

	// Collect findings per repo (only repos with changes)
	repoFindings := make(map[string][]output.FindingOutput)

	for _, repoPath := range repos {
		repoName := filepath.Base(repoPath)

		if !config.RepoConfigExists(repoPath) {
			continue
		}

		repoCfg, err := config.LoadRepoConfig(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s: %v\n", repoName, err)
			continue
		}

		validationErrors := schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath)
		if len(validationErrors) > 0 {
			fmt.Fprintf(os.Stderr, "  warning: %s: %d validation error(s)\n", repoName, len(validationErrors))
			continue
		}

		findings, err := engine.ComputeFindings(repoCfg, centralCfg, repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s: %v\n", repoName, err)
			continue
		}

		if len(findings) == 0 {
			continue
		}

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
		repoFindings[repoName] = findingOutputs
	}

	markdown := output.GenerateMarkdownReport(output.MarkdownReportInput{
		RepoFindings:  repoFindings,
		CollapseDiffs: rootCfg.Report.CollapseDiffs,
	})

	if *outputFlag != "" {
		if err := writeReportFile(*outputFlag, markdown); err != nil {
			output.Error(fmt.Sprintf("failed to write report: %v", err))
			os.Exit(1)
		}
	} else {
		fmt.Print(markdown)
	}
}

func writeReportFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}
