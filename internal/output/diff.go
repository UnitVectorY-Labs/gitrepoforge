package output

import (
	"fmt"
	"strings"
)

type diffOp struct {
	kind string
	line string
}

const maxDiffMatrixCells uint64 = 1 << 22

// RenderDiff returns a colorized git-style line diff for a finding.
func RenderDiff(f FindingOutput) []string {
	if f.Operation != "create" && f.Operation != "update" && f.Operation != "delete" {
		return nil
	}

	oldLabel := fmt.Sprintf("a/%s", f.FilePath)
	newLabel := fmt.Sprintf("b/%s", f.FilePath)
	if f.Operation == "create" {
		oldLabel = "/dev/null"
	}
	if f.Operation == "delete" {
		newLabel = "/dev/null"
	}

	lines := []string{
		fmt.Sprintf("%sdiff --git a/%s b/%s%s", Bold, f.FilePath, f.FilePath, Reset),
		fmt.Sprintf("%s--- %s%s", Cyan, oldLabel, Reset),
		fmt.Sprintf("%s+++ %s%s", Cyan, newLabel, Reset),
	}

	for _, op := range diffLines(f.Actual, f.Expected) {
		switch op.kind {
		case "equal":
			lines = append(lines, " "+op.line)
		case "delete":
			lines = append(lines, fmt.Sprintf("%s-%s%s", Red, op.line, Reset))
		case "insert":
			lines = append(lines, fmt.Sprintf("%s+%s%s", Green, op.line, Reset))
		}
	}

	return lines
}

func diffLines(actual, expected string) []diffOp {
	oldLines := splitLines(actual)
	newLines := splitLines(expected)

	rows, cols, ok := safeDiffMatrixDimensions(len(oldLines), len(newLines))
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

	var ops []diffOp
	i, j := 0, 0
	for i < len(oldLines) && j < len(newLines) {
		if oldLines[i] == newLines[j] {
			ops = append(ops, diffOp{kind: "equal", line: oldLines[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{kind: "delete", line: oldLines[i]})
			i++
		} else {
			ops = append(ops, diffOp{kind: "insert", line: newLines[j]})
			j++
		}
	}

	for ; i < len(oldLines); i++ {
		ops = append(ops, diffOp{kind: "delete", line: oldLines[i]})
	}
	for ; j < len(newLines); j++ {
		ops = append(ops, diffOp{kind: "insert", line: newLines[j]})
	}

	return ops
}

func safeDiffMatrixDimensions(oldLen, newLen int) (int, int, bool) {
	const maxInt = int(^uint(0) >> 1)

	if oldLen < 0 || newLen < 0 || oldLen >= maxInt || newLen >= maxInt {
		return 0, 0, false
	}

	rows := oldLen + 1
	cols := newLen + 1
	if uint64(rows)*uint64(cols) > maxDiffMatrixCells {
		return 0, 0, false
	}

	return rows, cols, true
}

func fallbackDiffLines(oldLines, newLines []string) []diffOp {
	ops := make([]diffOp, 0, len(oldLines)+len(newLines))
	for _, line := range oldLines {
		ops = append(ops, diffOp{kind: "delete", line: line})
	}
	for _, line := range newLines {
		ops = append(ops, diffOp{kind: "insert", line: line})
	}
	return ops
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
