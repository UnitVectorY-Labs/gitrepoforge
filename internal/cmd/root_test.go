package cmd

import (
	"fmt"
	"runtime"
	"testing"
)

func TestFormatVersionOutput(t *testing.T) {
	t.Parallel()

	got := formatVersionOutput("v1.2.3")
	want := fmt.Sprintf("gitrepoforge version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if got != want {
		t.Fatalf("formatVersionOutput() = %q, want %q", got, want)
	}
}
