package output

import (
	"strings"
	"testing"
)

func TestGenerateMarkdownReport_NoChanges(t *testing.T) {
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings:  map[string][]FindingOutput{},
		CollapseDiffs: true,
	})

	if !strings.Contains(result, "No changes detected") {
		t.Errorf("expected 'No changes detected' in output, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_SingleRepo(t *testing.T) {
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"my-repo": {
				{
					FilePath:  ".gitignore",
					Operation: "create",
					Message:   "file should exist",
					Expected:  "node_modules/\n.env\n",
					Actual:    "",
				},
			},
		},
		CollapseDiffs: true,
	})

	// Check header
	if !strings.Contains(result, "# gitrepoforge Report") {
		t.Errorf("expected report header, got:\n%s", result)
	}

	// Check repo summary table
	if !strings.Contains(result, "| my-repo | 1 |") {
		t.Errorf("expected repo summary row, got:\n%s", result)
	}

	// Check file summary table
	if !strings.Contains(result, "| `.gitignore` | create | 1 |") {
		t.Errorf("expected file summary row, got:\n%s", result)
	}

	// Check file section heading
	if !strings.Contains(result, "## `.gitignore` (create)") {
		t.Errorf("expected file section heading, got:\n%s", result)
	}

	// Check collapsed diff
	if !strings.Contains(result, "<details>") {
		t.Errorf("expected collapsed diff block, got:\n%s", result)
	}
	if !strings.Contains(result, "```diff") {
		t.Errorf("expected diff code block, got:\n%s", result)
	}
	if !strings.Contains(result, "+node_modules/") {
		t.Errorf("expected diff content with + prefix, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_CollapsedFalse(t *testing.T) {
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"my-repo": {
				{
					FilePath:  ".gitignore",
					Operation: "create",
					Message:   "file should exist",
					Expected:  "node_modules/\n",
					Actual:    "",
				},
			},
		},
		CollapseDiffs: false,
	})

	// Should NOT have details tag
	if strings.Contains(result, "<details>") {
		t.Errorf("expected no collapsed block when CollapseDiffs is false, got:\n%s", result)
	}

	// Should still have diff block
	if !strings.Contains(result, "```diff") {
		t.Errorf("expected diff code block, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_DeduplicatedDiffs(t *testing.T) {
	// Two repos with the same change to .gitignore
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"repo-a": {
				{
					FilePath:  ".gitignore",
					Operation: "create",
					Message:   "file should exist",
					Expected:  "node_modules/\n",
					Actual:    "",
				},
			},
			"repo-b": {
				{
					FilePath:  ".gitignore",
					Operation: "create",
					Message:   "file should exist",
					Expected:  "node_modules/\n",
					Actual:    "",
				},
			},
		},
		CollapseDiffs: true,
	})

	// Both repos listed in section
	if !strings.Contains(result, "repo-a") || !strings.Contains(result, "repo-b") {
		t.Errorf("expected both repos listed, got:\n%s", result)
	}

	// Should only have ONE diff block (deduplicated)
	count := strings.Count(result, "```diff")
	if count != 1 {
		t.Errorf("expected 1 deduplicated diff block, got %d in:\n%s", count, result)
	}
}

func TestGenerateMarkdownReport_DifferentDiffsSameFile(t *testing.T) {
	// Two repos with different changes to the same file
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"repo-a": {
				{
					FilePath:  ".gitignore",
					Operation: "update",
					Message:   "file needs updating",
					Expected:  "node_modules/\n.env\n",
					Actual:    "node_modules/\n",
				},
			},
			"repo-b": {
				{
					FilePath:  ".gitignore",
					Operation: "update",
					Message:   "file needs updating",
					Expected:  "node_modules/\n.build/\n",
					Actual:    "node_modules/\n",
				},
			},
		},
		CollapseDiffs: true,
	})

	// Should have TWO diff blocks (not deduplicated since changes differ)
	count := strings.Count(result, "```diff")
	if count != 2 {
		t.Errorf("expected 2 diff blocks for different changes, got %d in:\n%s", count, result)
	}

	// Should have change sub-headings
	if !strings.Contains(result, "### Change 1") || !strings.Contains(result, "### Change 2") {
		t.Errorf("expected change sub-headings, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_MultipleFiles(t *testing.T) {
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"my-repo": {
				{
					FilePath:  ".gitignore",
					Operation: "create",
					Message:   "file should exist",
					Expected:  "node_modules/\n",
					Actual:    "",
				},
				{
					FilePath:  "LICENSE",
					Operation: "update",
					Message:   "file needs updating",
					Expected:  "MIT License\n",
					Actual:    "Old License\n",
				},
			},
		},
		CollapseDiffs: true,
	})

	// Check both file sections exist
	if !strings.Contains(result, "## `.gitignore` (create)") {
		t.Errorf("expected .gitignore section, got:\n%s", result)
	}
	if !strings.Contains(result, "## `LICENSE` (update)") {
		t.Errorf("expected LICENSE section, got:\n%s", result)
	}

	// Check file summary table has both
	if !strings.Contains(result, "| `.gitignore` | create | 1 |") {
		t.Errorf("expected .gitignore in file summary, got:\n%s", result)
	}
	if !strings.Contains(result, "| `LICENSE` | update | 1 |") {
		t.Errorf("expected LICENSE in file summary, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_DeleteOperation(t *testing.T) {
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"my-repo": {
				{
					FilePath:  "old-file.txt",
					Operation: "delete",
					Message:   "file should not exist",
					Expected:  "",
					Actual:    "old content\n",
				},
			},
		},
		CollapseDiffs: true,
	})

	if !strings.Contains(result, "## `old-file.txt` (delete)") {
		t.Errorf("expected delete section, got:\n%s", result)
	}
	if !strings.Contains(result, "-old content") {
		t.Errorf("expected deleted content in diff, got:\n%s", result)
	}
}

func TestGenerateMarkdownReport_DeduplicateIgnoresContext(t *testing.T) {
	// Two repos with the same change (+.env) but different surrounding context
	result := GenerateMarkdownReport(MarkdownReportInput{
		RepoFindings: map[string][]FindingOutput{
			"repo-a": {
				{
					FilePath:  ".gitignore",
					Operation: "update",
					Message:   "file needs updating",
					Expected:  "node_modules/\n.env\n",
					Actual:    "node_modules/\n",
				},
			},
			"repo-b": {
				{
					FilePath:  ".gitignore",
					Operation: "update",
					Message:   "file needs updating",
					Expected:  "build/\n.env\n",
					Actual:    "build/\n",
				},
			},
		},
		CollapseDiffs: true,
	})

	// Same change (+.env) but different context - should be deduplicated to 1 diff
	count := strings.Count(result, "```diff")
	if count != 1 {
		t.Errorf("expected 1 deduplicated diff block (same change, different context), got %d in:\n%s", count, result)
	}
}

func TestRenderPlainDiff_Create(t *testing.T) {
	diff := renderPlainDiff(FindingOutput{
		FilePath:  ".gitignore",
		Operation: "create",
		Expected:  "node_modules/\n",
		Actual:    "",
	})

	if !strings.Contains(diff, "--- /dev/null") {
		t.Errorf("expected '--- /dev/null' for create, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ b/.gitignore") {
		t.Errorf("expected '+++ b/.gitignore' for create, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+node_modules/") {
		t.Errorf("expected '+node_modules/' in diff, got:\n%s", diff)
	}
}

func TestRenderPlainDiff_Delete(t *testing.T) {
	diff := renderPlainDiff(FindingOutput{
		FilePath:  "old-file.txt",
		Operation: "delete",
		Expected:  "",
		Actual:    "old content\n",
	})

	if !strings.Contains(diff, "--- a/old-file.txt") {
		t.Errorf("expected '--- a/old-file.txt' for delete, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ /dev/null") {
		t.Errorf("expected '+++ /dev/null' for delete, got:\n%s", diff)
	}
	if !strings.Contains(diff, "-old content") {
		t.Errorf("expected '-old content' in diff, got:\n%s", diff)
	}
}

func TestExtractChangeLines(t *testing.T) {
	diff := " context line\n+added line\n-removed line\n another context"
	result := extractChangeLines(diff)
	if result != "+added line\n-removed line" {
		t.Errorf("expected only change lines, got: %q", result)
	}
}
