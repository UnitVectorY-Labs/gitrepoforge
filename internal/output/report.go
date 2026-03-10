package output

import (
	"encoding/json"
	"fmt"
	"time"
)

// Report is the structured JSON output for all commands.
type Report struct {
	Tool       ToolMeta     `json:"tool"`
	RootConfig string       `json:"root_config_path"`
	ConfigRepo string       `json:"central_config_path"`
	Repos      []RepoResult `json:"repos"`
}

// ToolMeta contains tool metadata.
type ToolMeta struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Command   string `json:"command"`
}

// RepoResult represents the result for a single repository.
type RepoResult struct {
	Name             string           `json:"repo"`
	Status           string           `json:"status"`
	ValidationErrors []string         `json:"validation_errors,omitempty"`
	Findings         []FindingOutput  `json:"findings,omitempty"`
}

// FindingOutput is the JSON representation of a finding.
type FindingOutput struct {
	FilePath  string `json:"file_path"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

// NewReport creates a new report.
func NewReport(version, command, rootConfigPath, configRepoPath string) *Report {
	return &Report{
		Tool: ToolMeta{
			Name:      "gitrepoforge",
			Version:   version,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Command:   command,
		},
		RootConfig: rootConfigPath,
		ConfigRepo: configRepoPath,
	}
}

// PrintJSON outputs the report as JSON.
func (r *Report) PrintJSON() error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintHuman outputs the report in human-readable format.
func (r *Report) PrintHuman() {
	for _, repo := range r.Repos {
		switch repo.Status {
		case "clean":
			Success(fmt.Sprintf("%s: compliant", repo.Name))
		case "skipped":
			Warning(fmt.Sprintf("%s: skipped (no .gitrepoforge file)", repo.Name))
		case "invalid":
			Error(fmt.Sprintf("%s: invalid configuration", repo.Name))
			for _, e := range repo.ValidationErrors {
				Detail(fmt.Sprintf("  %s", e))
			}
		case "drift":
			Warning(fmt.Sprintf("%s: non-compliant (%d findings)", repo.Name, len(repo.Findings)))
			for _, f := range repo.Findings {
				Detail(fmt.Sprintf("  [%s] %s: %s", f.Operation, f.FilePath, f.Message))
			}
		case "applied":
			Success(fmt.Sprintf("%s: changes applied", repo.Name))
			for _, f := range repo.Findings {
				Detail(fmt.Sprintf("  [%s] %s: %s", f.Operation, f.FilePath, f.Message))
			}
		case "failed":
			Error(fmt.Sprintf("%s: failed", repo.Name))
			for _, e := range repo.ValidationErrors {
				Detail(fmt.Sprintf("  %s", e))
			}
		default:
			Info(fmt.Sprintf("%s: %s", repo.Name, repo.Status))
		}
	}
}

// HasFailures returns true if any repo has a non-clean status that indicates failure.
func (r *Report) HasFailures() bool {
	for _, repo := range r.Repos {
		switch repo.Status {
		case "invalid", "drift", "failed":
			return true
		}
	}
	return false
}
