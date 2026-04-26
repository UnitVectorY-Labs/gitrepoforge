package output

import (
	"strings"
	"testing"
)

func TestRenderDiffUpdate(t *testing.T) {
	lines := RenderDiff(FindingOutput{
		FilePath:  "README.md",
		Operation: "update",
		Actual:    "one\ntwo\n",
		Expected:  "one\nthree\n",
	})

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "diff --git a/README.md b/README.md") {
		t.Fatalf("missing diff header: %q", joined)
	}
	if !strings.Contains(joined, "-two") {
		t.Fatalf("missing removed line: %q", joined)
	}
	if !strings.Contains(joined, "+three") {
		t.Fatalf("missing added line: %q", joined)
	}
}

func TestRenderDiffCreate(t *testing.T) {
	lines := RenderDiff(FindingOutput{
		FilePath:  "justfile",
		Operation: "create",
		Expected:  "default:\n  @just --list\n",
	})

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "--- /dev/null") {
		t.Fatalf("missing create old header: %q", joined)
	}
	if !strings.Contains(joined, "+++ b/justfile") {
		t.Fatalf("missing create new header: %q", joined)
	}
	if !strings.Contains(joined, "+default:") {
		t.Fatalf("missing added content: %q", joined)
	}
}

func TestRenderDiffDelete(t *testing.T) {
	lines := RenderDiff(FindingOutput{
		FilePath:  "obsolete.txt",
		Operation: "delete",
		Actual:    "remove me\n",
	})

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "--- a/obsolete.txt") {
		t.Fatalf("missing delete old header: %q", joined)
	}
	if !strings.Contains(joined, "+++ /dev/null") {
		t.Fatalf("missing delete new header: %q", joined)
	}
	if !strings.Contains(joined, "-remove me") {
		t.Fatalf("missing removed content: %q", joined)
	}
}

func TestSafeDiffMatrixDimensionsRejectsOversizedAllocation(t *testing.T) {
	rows, cols, ok := safeDiffMatrixDimensions(2048, 2048)
	if ok {
		t.Fatalf("expected oversized diff matrix to be rejected, got rows=%d cols=%d", rows, cols)
	}
}

func TestFallbackDiffLinesPreservesAllLines(t *testing.T) {
	ops := fallbackDiffLines([]string{"old-1", "old-2"}, []string{"new-1"})
	if len(ops) != 3 {
		t.Fatalf("len(ops) = %d, want 3", len(ops))
	}
	if ops[0].kind != "delete" || ops[0].line != "old-1" {
		t.Fatalf("ops[0] = %+v, want delete old-1", ops[0])
	}
	if ops[1].kind != "delete" || ops[1].line != "old-2" {
		t.Fatalf("ops[1] = %+v, want delete old-2", ops[1])
	}
	if ops[2].kind != "insert" || ops[2].line != "new-1" {
		t.Fatalf("ops[2] = %+v, want insert new-1", ops[2])
	}
}
