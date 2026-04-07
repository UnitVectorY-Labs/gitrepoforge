package output

import (
	"fmt"
	"sort"
	"strings"
)

// MarkdownReportInput holds the data needed to generate a markdown report.
type MarkdownReportInput struct {
	// RepoFindings maps repo name to its list of findings.
	RepoFindings map[string][]FindingOutput
	// CollapseDiffs controls whether diffs are wrapped in <details> blocks.
	CollapseDiffs bool
}

// fileChangeGroup represents a group of changes for a single output file path.
type fileChangeGroup struct {
	FilePath   string
	Operation  string
	RepoNames  []string
	UniqueDiff []uniqueDiffEntry
}

// uniqueDiffEntry represents one deduplicated diff with the repos that share it.
type uniqueDiffEntry struct {
	DiffLines string
	RepoNames []string
}

// repoFinding pairs a repo name with one of its findings.
type repoFinding struct {
	repoName string
	finding  FindingOutput
}

// GenerateMarkdownReport produces a markdown report string from the input.
func GenerateMarkdownReport(input MarkdownReportInput) string {
	if len(input.RepoFindings) == 0 {
		return "# gitrepoforge Report\n\nNo changes detected.\n"
	}

	// Collect all findings grouped by file path
	groups := buildFileChangeGroups(input.RepoFindings)

	var sb strings.Builder

	sb.WriteString("# gitrepoforge Report\n\n")

	// Summary table: repos with changes
	writeRepoSummaryTable(&sb, input.RepoFindings)

	// Summary table: files with changes
	writeFileSummaryTable(&sb, groups)

	// Detailed sections per file path
	for _, group := range groups {
		writeFileSection(&sb, group, input.CollapseDiffs)
	}

	return sb.String()
}

// buildFileChangeGroups aggregates findings by file path, deduplicating diffs.
func buildFileChangeGroups(repoFindings map[string][]FindingOutput) []fileChangeGroup {
	type fileKey struct {
		path      string
		operation string
	}

	// Gather findings by (path, operation)
	grouped := make(map[fileKey][]repoFinding)

	for repoName, findings := range repoFindings {
		for _, f := range findings {
			key := fileKey{path: f.FilePath, operation: f.Operation}
			grouped[key] = append(grouped[key], repoFinding{repoName: repoName, finding: f})
		}
	}

	// Sort keys for deterministic output
	var keys []fileKey
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].path != keys[j].path {
			return keys[i].path < keys[j].path
		}
		return keys[i].operation < keys[j].operation
	})

	var groups []fileChangeGroup
	for _, key := range keys {
		rfs := grouped[key]

		// Collect all repo names
		var repoNames []string
		for _, rf := range rfs {
			repoNames = append(repoNames, rf.repoName)
		}
		sort.Strings(repoNames)

		// Deduplicate diffs: compute diff for each, group by diff content
		uniqueDiffs := deduplicateDiffs(rfs)

		groups = append(groups, fileChangeGroup{
			FilePath:   key.path,
			Operation:  key.operation,
			RepoNames:  repoNames,
			UniqueDiff: uniqueDiffs,
		})
	}

	return groups
}

// deduplicateDiffs computes the diff for each repo's finding and groups by
// identical change lines (ignoring context lines) to deduplicate.
func deduplicateDiffs(rfs []repoFinding) []uniqueDiffEntry {
	type diffGroup struct {
		changeKey string
		fullDiff  string
		repos     []string
	}

	var groups []diffGroup
	keyIndex := make(map[string]int)

	for _, rf := range rfs {
		diffStr := renderPlainDiff(rf.finding)
		changeKey := extractChangeLines(diffStr)

		if idx, ok := keyIndex[changeKey]; ok {
			groups[idx].repos = append(groups[idx].repos, rf.repoName)
		} else {
			keyIndex[changeKey] = len(groups)
			groups = append(groups, diffGroup{
				changeKey: changeKey,
				fullDiff:  diffStr,
				repos:     []string{rf.repoName},
			})
		}
	}

	var result []uniqueDiffEntry
	for _, g := range groups {
		sort.Strings(g.repos)
		result = append(result, uniqueDiffEntry{
			DiffLines: g.fullDiff,
			RepoNames: g.repos,
		})
	}

	return result
}

// extractChangeLines extracts only the added/removed lines from a diff string
// to use as a deduplication key. This ignores context lines which may differ
// between repos.
func extractChangeLines(diff string) string {
	lines := strings.Split(diff, "\n")
	var changes []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			changes = append(changes, line)
		}
	}
	return strings.Join(changes, "\n")
}

// renderPlainDiff produces a plain-text unified diff (no ANSI colors) for a finding.
func renderPlainDiff(f FindingOutput) string {
	if f.Operation != "create" && f.Operation != "update" && f.Operation != "delete" {
		return ""
	}

	oldLabel := fmt.Sprintf("a/%s", f.FilePath)
	newLabel := fmt.Sprintf("b/%s", f.FilePath)
	if f.Operation == "create" {
		oldLabel = "/dev/null"
	}
	if f.Operation == "delete" {
		newLabel = "/dev/null"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("--- %s", oldLabel))
	lines = append(lines, fmt.Sprintf("+++ %s", newLabel))

	for _, op := range diffLines(f.Actual, f.Expected) {
		switch op.kind {
		case "equal":
			lines = append(lines, " "+op.line)
		case "delete":
			lines = append(lines, "-"+op.line)
		case "insert":
			lines = append(lines, "+"+op.line)
		}
	}

	return strings.Join(lines, "\n")
}

func writeRepoSummaryTable(sb *strings.Builder, repoFindings map[string][]FindingOutput) {
	sb.WriteString("## Repository Summary\n\n")
	sb.WriteString("| Repository | Changes |\n")
	sb.WriteString("|---|---|\n")

	// Sort repo names for deterministic output
	var repoNames []string
	for name := range repoFindings {
		repoNames = append(repoNames, name)
	}
	sort.Strings(repoNames)

	for _, name := range repoNames {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", name, len(repoFindings[name])))
	}
	sb.WriteString("\n")
}

func writeFileSummaryTable(sb *strings.Builder, groups []fileChangeGroup) {
	sb.WriteString("## File Summary\n\n")
	sb.WriteString("| File | Operation | Repositories |\n")
	sb.WriteString("|---|---|---|\n")

	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %d |\n", g.FilePath, g.Operation, len(g.RepoNames)))
	}
	sb.WriteString("\n")
}

func writeFileSection(sb *strings.Builder, group fileChangeGroup, collapseDiffs bool) {
	sb.WriteString(fmt.Sprintf("## `%s` (%s)\n\n", group.FilePath, group.Operation))

	sb.WriteString("**Repositories:** ")
	sb.WriteString(strings.Join(group.RepoNames, ", "))
	sb.WriteString("\n\n")

	for i, entry := range group.UniqueDiff {
		if len(group.UniqueDiff) > 1 {
			sb.WriteString(fmt.Sprintf("### Change %d\n\n", i+1))
			sb.WriteString("**Repositories:** ")
			sb.WriteString(strings.Join(entry.RepoNames, ", "))
			sb.WriteString("\n\n")
		}

		if entry.DiffLines == "" {
			continue
		}

		if collapseDiffs {
			sb.WriteString("<details>\n")
			sb.WriteString("<summary>Diff</summary>\n\n")
		}

		sb.WriteString("```diff\n")
		sb.WriteString(entry.DiffLines)
		sb.WriteString("\n```\n")

		if collapseDiffs {
			sb.WriteString("\n</details>\n")
		}

		sb.WriteString("\n")
	}
}
