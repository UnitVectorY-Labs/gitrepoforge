package diff

import (
	"strings"
	"testing"
)

func TestRenderUnifiedUpdateUsesGitStyleHunks(t *testing.T) {
	actual := strings.Join([]string{
		"",
		"# gitrepoforge",
		"",
		"Command line application to audit and apply standard file patterns to Git repos.",
		"",
	}, "\n") + "\n"
	expected := strings.Join([]string{
		"",
		"# gitrepoforge",
		"",
		"This is a test.",
		"",
		"Command line application to audit and apply standard file patterns to Git repos.",
		"",
		"Showing changes",
		"",
	}, "\n") + "\n"

	lines := RenderUnified(Options{
		Path:      "README.md",
		Operation: "update",
		Actual:    actual,
		Expected:  expected,
		Context:   3,
	})

	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"diff --git a/README.md b/README.md",
		"--- a/README.md",
		"+++ b/README.md",
		"@@ -1,5 +1,9 @@",
		" ",
		" # gitrepoforge",
		" ",
		"+This is a test.",
		"+",
		" Command line application to audit and apply standard file patterns to Git repos.",
		" ",
		"+Showing changes",
		"+",
	}, "\n")
	if got != want {
		t.Fatalf("diff mismatch\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestRenderUnifiedLimitsUnchangedContext(t *testing.T) {
	actual := "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\n"
	expected := "one\ntwo\nthree\nFOUR\nfive\nsix\nseven\neight\nnine\n"

	got := strings.Join(RenderUnified(Options{
		Path:      "numbers.txt",
		Operation: "update",
		Actual:    actual,
		Expected:  expected,
		Context:   1,
	}), "\n")

	if strings.Contains(got, " one") || strings.Contains(got, " nine") {
		t.Fatalf("diff included context outside requested hunk:\n%s", got)
	}
	if !strings.Contains(got, "@@ -3,3 +3,3 @@") {
		t.Fatalf("missing focused hunk header:\n%s", got)
	}
}

func TestRenderUnifiedCreateAndDeleteHeaders(t *testing.T) {
	create := strings.Join(RenderUnified(Options{
		Path:      ".gitignore",
		Operation: "create",
		Expected:  "node_modules/\n",
		Context:   3,
	}), "\n")
	if !strings.Contains(create, "--- /dev/null") || !strings.Contains(create, "+++ b/.gitignore") {
		t.Fatalf("create headers mismatch:\n%s", create)
	}
	if !strings.Contains(create, "@@ -0,0 +1 @@") {
		t.Fatalf("create hunk header mismatch:\n%s", create)
	}

	deleteDiff := strings.Join(RenderUnified(Options{
		Path:      "old-file.txt",
		Operation: "delete",
		Actual:    "old content\n",
		Context:   3,
	}), "\n")
	if !strings.Contains(deleteDiff, "--- a/old-file.txt") || !strings.Contains(deleteDiff, "+++ /dev/null") {
		t.Fatalf("delete headers mismatch:\n%s", deleteDiff)
	}
	if !strings.Contains(deleteDiff, "@@ -1 +0,0 @@") {
		t.Fatalf("delete hunk header mismatch:\n%s", deleteDiff)
	}
}

func TestRenderUnifiedColor(t *testing.T) {
	got := strings.Join(RenderUnified(Options{
		Path:      "README.md",
		Operation: "update",
		Actual:    "old\n",
		Expected:  "new\n",
		Context:   3,
		Color:     true,
	}), "\n")
	if !strings.Contains(got, "\033[31m-old\033[0m") {
		t.Fatalf("missing colorized delete line:\n%q", got)
	}
	if !strings.Contains(got, "\033[32m+new\033[0m") {
		t.Fatalf("missing colorized insert line:\n%q", got)
	}
}

func TestSafeMatrixDimensionsRejectsOversizedAllocation(t *testing.T) {
	rows, cols, ok := safeMatrixDimensions(2048, 2048)
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

func TestSafeCombinedLengthRejectsOverflow(t *testing.T) {
	const maxInt = int(^uint(0) >> 1)
	if got := safeCombinedLength(maxInt, 1); got != 0 {
		t.Fatalf("safeCombinedLength(maxInt, 1) = %d, want 0", got)
	}
}
