package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	actionFlag := fs.String("action", "", "Named action from the action config to use for git automation")
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
	report.IgnoreMissing = rootCfg.IgnoreMissing

	gitCfg, actionName, err := resolveApplyAction(rootCfg, *actionFlag, flagPassed(fs, "action"))
	if err != nil {
		output.Error(fmt.Sprintf("action error: %v", err))
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

	if !*jsonFlag {
		output.Header("gitrepoforge apply")
		fmt.Println()
	}

	for _, repoPath := range repos {
		repoName := filepath.Base(repoPath)
		result := applyRepo(repoPath, repoName, rootCfg, gitCfg, actionName, centralCfg)
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

func applyRepo(repoPath, repoName string, rootCfg *config.RootConfig, gitCfg *config.GitConfig, actionName string, centralCfg *config.CentralConfig) output.RepoResult {
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

	if gitValidationErrors := validateRootGitTemplates(gitCfg, actionName, repoCfg); len(gitValidationErrors) > 0 {
		return output.RepoResult{
			Name:             repoName,
			Status:           "invalid",
			ValidationErrors: gitValidationErrors,
		}
	}

	findings, err := engine.ComputeFindings(repoCfg, centralCfg, repoPath, config.ResolveManifestPath(rootCfg, repoCfg))
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
		if !gitCfg.Commit {
			result.StatusDetail = cleanStatusDetail(repoPath)
		}
		return result
	}

	if actionName == "" {
		return output.RepoResult{
			Name:     repoName,
			Status:   "drift",
			Findings: findingsToOutput(findings),
		}
	}

	return applyFindingsWithGit(repoPath, repoName, repoCfg, gitCfg, findings)
}

func applyFindingsWithGit(repoPath, repoName string, repoCfg *config.RepoConfig, gitCfg *config.GitConfig, findings []engine.Finding) output.RepoResult {
	findingOutputs := findingsToOutput(findings)

	gitEnabled := gitCfg.GitOptionsSpecified()
	placeholderValues := repoCfg.PlaceholderValues()

	originalBranch := ""
	branchName := ""
	createdBranch := false

	restoreOriginalBranch := func() {
		if !gitEnabled || !gitCfg.ReturnToOriginalBranch || !createdBranch || originalBranch == "" {
			return
		}
		_ = gitops.CheckoutBranch(repoPath, originalBranch)
	}

	if gitEnabled {
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
				ValidationErrors: []string{"repo has uncommitted changes; git automation requires a clean working tree"},
			}
		}

		originalBranch, err = gitops.CurrentBranch(repoPath)
		if err != nil {
			return output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{fmt.Sprintf("failed to determine current branch: %v", err)},
			}
		}
		branchName = originalBranch

		if gitCfg.CreateBranch {
			branchName = gitCfg.BuildBranchName(placeholderValues)
			if err := gitops.CreateBranch(repoPath, branchName); err != nil {
				return output.RepoResult{
					Name:             repoName,
					Status:           "failed",
					ValidationErrors: []string{fmt.Sprintf("failed to create branch: %v", err)},
				}
			}
			createdBranch = true
		}
	}

	if err := engine.ApplyFindings(findings, repoPath); err != nil {
		restoreOriginalBranch()
		return output.RepoResult{
			Name:             repoName,
			Status:           "failed",
			ValidationErrors: []string{fmt.Sprintf("failed to apply changes: %v", err)},
		}
	}

	if gitCfg.Commit {
		if err := gitops.AddAll(repoPath); err != nil {
			restoreOriginalBranch()
			return output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{fmt.Sprintf("failed to stage changes: %v", err)},
			}
		}

		hasChanges, err := gitops.HasChanges(repoPath)
		if err != nil {
			restoreOriginalBranch()
			return output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{fmt.Sprintf("failed to check for changes: %v", err)},
			}
		}
		if !hasChanges {
			restoreOriginalBranch()
			return output.RepoResult{
				Name:   repoName,
				Status: "clean",
			}
		}

		commitMessage := gitCfg.BuildCommitMessage(placeholderValues)
		if err := gitops.Commit(repoPath, commitMessage); err != nil {
			restoreOriginalBranch()
			return output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{fmt.Sprintf("failed to commit: %v", err)},
			}
		}

		if gitCfg.Push {
			if err := gitops.Push(repoPath, gitCfg.Remote, branchName); err != nil {
				restoreOriginalBranch()
				return output.RepoResult{
					Name:             repoName,
					Status:           "failed",
					ValidationErrors: []string{fmt.Sprintf("failed to push: %v", err)},
				}
			}

			if gitCfg.PullRequest == config.PullRequestGitHubCLI {
				if err := gitops.CreatePR(repoPath); err != nil {
					output.Warning(fmt.Sprintf("%s: PR creation failed: %v", repoName, err))
				}
			}
		}
	}

	if gitEnabled && gitCfg.ReturnToOriginalBranch {
		if err := gitops.CheckoutBranch(repoPath, originalBranch); err != nil {
			return output.RepoResult{
				Name:             repoName,
				Status:           "failed",
				ValidationErrors: []string{fmt.Sprintf("failed to return to original branch: %v", err)},
			}
		}
		if gitCfg.DeleteBranch && createdBranch {
			if err := gitops.DeleteBranch(repoPath, branchName); err != nil {
				output.Warning(fmt.Sprintf("%s: failed to delete branch %s: %v", repoName, branchName, err))
			}
		}
	}

	return output.RepoResult{
		Name:     repoName,
		Status:   "applied",
		Findings: findingOutputs,
	}
}

func validateRootGitTemplates(gitCfg *config.GitConfig, actionName string, repoCfg *config.RepoConfig) []string {
	if actionName == "" {
		return nil
	}
	var errors []string
	values := repoCfg.PlaceholderValues()
	prefix := "action." + actionName

	if gitCfg.CreateBranch {
		errors = append(errors, validateGitTemplate(prefix+".branch_name", gitCfg.BranchName, values)...)
	}
	if gitCfg.Commit {
		errors = append(errors, validateGitTemplate(prefix+".commit_message", gitCfg.CommitMessage, values)...)
	}

	return errors
}

func validateGitTemplate(field, template string, values map[string]string) []string {
	placeholders := config.ExtractGitPlaceholders(template)
	if len(placeholders) == 0 {
		return nil
	}

	var unknown []string
	for _, placeholder := range placeholders {
		if _, ok := values[placeholder]; ok {
			continue
		}
		unknown = append(unknown, placeholder)
	}
	if len(unknown) == 0 {
		return nil
	}

	sort.Strings(unknown)
	return []string{fmt.Sprintf("%s: unknown placeholder(s): %s", field, strings.Join(unknown, ", "))}
}

func findingsToOutput(findings []engine.Finding) []output.FindingOutput {
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
	return findingOutputs
}

func flagPassed(fs *flag.FlagSet, name string) bool {
	passed := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			passed = true
		}
	})
	return passed
}

func resolveApplyAction(rootCfg *config.RootConfig, actionName string, actionSpecified bool) (*config.GitConfig, string, error) {
	if !actionSpecified {
		return &config.GitConfig{}, "", nil
	}
	if actionName == "" {
		return nil, "", fmt.Errorf("--action requires a configured action%s", configuredActionSuffix(rootCfg))
	}

	gitCfg, err := rootCfg.ResolveAction(actionName)
	if err != nil {
		return nil, "", fmt.Errorf("%v%s", err, configuredActionSuffix(rootCfg))
	}
	return gitCfg, actionName, nil
}

func configuredActionSuffix(rootCfg *config.RootConfig) string {
	if len(rootCfg.Actions) == 0 {
		return " (no actions are configured)"
	}

	names := make([]string, 0, len(rootCfg.Actions))
	for name := range rootCfg.Actions {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Sprintf(" (available actions: %s)", strings.Join(names, ", "))
}

// cleanStatusDetail checks the git status of a repo that is already compliant
// and returns a detail string indicating whether changes are staged or unstaged.
// Returns "" if the repo is fully clean, "staged" if there are staged but
// uncommitted changes, or "unstaged" if there are unstaged changes.
func cleanStatusDetail(repoPath string) string {
	clean, err := gitops.IsClean(repoPath)
	if err != nil || clean {
		return ""
	}
	staged, err := gitops.HasChanges(repoPath)
	if err != nil {
		return ""
	}
	if staged {
		return "staged"
	}
	return "unstaged"
}
