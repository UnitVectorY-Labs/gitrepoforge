package output

import (
	"os"
	"strings"

	unifieddiff "github.com/UnitVectorY-Labs/gitrepoforge/internal/diff"
)

const defaultDiffContext = 3

// RenderDiff returns a git-style unified diff for a finding.
func RenderDiff(f FindingOutput) []string {
	return renderDiff(f, os.Getenv("NO_COLOR") == "")
}

func renderDiff(f FindingOutput, color bool) []string {
	return unifieddiff.RenderUnified(unifieddiff.Options{
		Path:      f.FilePath,
		Operation: f.Operation,
		Actual:    f.Actual,
		Expected:  f.Expected,
		Context:   defaultDiffContext,
		Color:     color,
	})
}

// renderPlainDiff produces a plain-text unified diff (no ANSI colors) for a finding.
func renderPlainDiff(f FindingOutput) string {
	return strings.Join(renderDiff(f, false), "\n")
}
