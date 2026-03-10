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
