package diff

import (
	"fmt"
	"strings"
)

const maxMatrixCells uint64 = 1 << 22

// Options controls unified diff rendering.
type Options struct {
	Path      string
	Actual    string
	Expected  string
	Operation string
	Context   int
	Color     bool
}

type lineOp struct {
	kind    string
	line    string
	oldLine int
	newLine int
}

type hunk struct {
	start int
	end   int
}

// RenderUnified returns a git-style unified diff for a file change.
func RenderUnified(opts Options) []string {
	if opts.Operation != "create" && opts.Operation != "update" && opts.Operation != "delete" {
		return nil
	}

	context := opts.Context
	if context < 0 {
		context = 0
	}

	oldLabel := fmt.Sprintf("a/%s", opts.Path)
	newLabel := fmt.Sprintf("b/%s", opts.Path)
	if opts.Operation == "create" {
		oldLabel = "/dev/null"
	}
	if opts.Operation == "delete" {
		newLabel = "/dev/null"
	}

	style := styles{}
	if opts.Color {
		style = coloredStyles()
	}

	lines := []string{
		style.diffLine(fmt.Sprintf("diff --git a/%s b/%s", opts.Path, opts.Path)),
		style.fileLine(fmt.Sprintf("--- %s", oldLabel)),
		style.fileLine(fmt.Sprintf("+++ %s", newLabel)),
	}

	ops := diffLines(opts.Actual, opts.Expected)
	for _, h := range buildHunks(ops, context) {
		lines = append(lines, style.hunkLine(renderHunkHeader(ops[h.start:h.end])))
		for _, op := range ops[h.start:h.end] {
			switch op.kind {
			case "equal":
				lines = append(lines, " "+op.line)
			case "delete":
				lines = append(lines, style.deleteLine("-"+op.line))
			case "insert":
				lines = append(lines, style.insertLine("+"+op.line))
			}
		}
	}

	return lines
}

type styles struct {
	reset string
	bold  string
	red   string
	green string
	cyan  string
}

func coloredStyles() styles {
	return styles{
		reset: "\033[0m",
		bold:  "\033[1m",
		red:   "\033[31m",
		green: "\033[32m",
		cyan:  "\033[36m",
	}
}

func (s styles) diffLine(line string) string {
	return s.bold + line + s.reset
}

func (s styles) fileLine(line string) string {
	return s.cyan + line + s.reset
}

func (s styles) hunkLine(line string) string {
	return s.cyan + line + s.reset
}

func (s styles) deleteLine(line string) string {
	return s.red + line + s.reset
}

func (s styles) insertLine(line string) string {
	return s.green + line + s.reset
}

func renderHunkHeader(ops []lineOp) string {
	oldStart, oldCount := hunkRange(ops, true)
	newStart, newCount := hunkRange(ops, false)
	return fmt.Sprintf("@@ -%s +%s @@", formatRange(oldStart, oldCount), formatRange(newStart, newCount))
}

func hunkRange(ops []lineOp, old bool) (int, int) {
	start := 0
	count := 0
	for _, op := range ops {
		consumes := op.kind == "equal" || (old && op.kind == "delete") || (!old && op.kind == "insert")
		if !consumes {
			continue
		}
		if start == 0 {
			if old {
				start = op.oldLine
			} else {
				start = op.newLine
			}
		}
		count++
	}
	if count == 0 {
		for _, op := range ops {
			if old {
				return op.oldLine, 0
			}
			return op.newLine, 0
		}
	}
	return start, count
}

func formatRange(start, count int) string {
	if count == 1 {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

func buildHunks(ops []lineOp, context int) []hunk {
	var hunks []hunk
	for i, op := range ops {
		if op.kind == "equal" {
			continue
		}
		start := max(i-context, 0)
		end := min(i+context+1, len(ops))
		if len(hunks) > 0 && start <= hunks[len(hunks)-1].end {
			if end > hunks[len(hunks)-1].end {
				hunks[len(hunks)-1].end = end
			}
			continue
		}
		hunks = append(hunks, hunk{start: start, end: end})
	}
	return hunks
}

func diffLines(actual, expected string) []lineOp {
	oldLines := splitLines(actual)
	newLines := splitLines(expected)

	rows, cols, ok := safeMatrixDimensions(len(oldLines), len(newLines))
	if !ok {
		return fallbackDiffLines(oldLines, newLines)
	}

	dp := make([][]int, rows)
	for i := range dp {
		dp[i] = make([]int, cols)
	}

	for i := len(oldLines) - 1; i >= 0; i-- {
		for j := len(newLines) - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []lineOp
	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			ops = append(ops, lineOp{kind: "equal", line: oldLines[i], oldLine: i + 1, newLine: j + 1})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, lineOp{kind: "delete", line: oldLines[i], oldLine: i + 1, newLine: j})
			i++
		} else {
			ops = append(ops, lineOp{kind: "insert", line: newLines[j], oldLine: i, newLine: j + 1})
			j++
		}
	}

	for ; i < len(oldLines); i++ {
		ops = append(ops, lineOp{kind: "delete", line: oldLines[i], oldLine: i + 1, newLine: j})
	}
	for ; j < len(newLines); j++ {
		ops = append(ops, lineOp{kind: "insert", line: newLines[j], oldLine: i, newLine: j + 1})
	}

	return ops
}

func safeMatrixDimensions(oldLen, newLen int) (int, int, bool) {
	const maxInt = int(^uint(0) >> 1)

	if oldLen < 0 || newLen < 0 || oldLen >= maxInt || newLen >= maxInt {
		return 0, 0, false
	}

	rows := oldLen + 1
	cols := newLen + 1
	if uint64(rows)*uint64(cols) > maxMatrixCells {
		return 0, 0, false
	}

	return rows, cols, true
}

func fallbackDiffLines(oldLines, newLines []string) []lineOp {
	ops := make([]lineOp, 0, safeCombinedLength(len(oldLines), len(newLines)))
	for i, line := range oldLines {
		ops = append(ops, lineOp{kind: "delete", line: line, oldLine: i + 1, newLine: 0})
	}
	for j, line := range newLines {
		ops = append(ops, lineOp{kind: "insert", line: line, oldLine: len(oldLines), newLine: j + 1})
	}
	return ops
}

func safeCombinedLength(a, b int) int {
	const maxInt = int(^uint(0) >> 1)

	if a < 0 || b < 0 || a > maxInt-b {
		return 0
	}
	return a + b
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
