package cmd

import (
	"fmt"
	"runtime"
	"testing"
)

func TestFormatVersionOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "prefixed semver",
			in:   "v1.2.3",
			want: fmt.Sprintf("gitrepoforge version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			name: "unprefixed semver",
			in:   "1.2.3",
			want: fmt.Sprintf("gitrepoforge version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
		{
			name: "dev string",
			in:   "dev",
			want: fmt.Sprintf("gitrepoforge version dev (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatVersionOutput(tt.in)
			if got != tt.want {
				t.Fatalf("formatVersionOutput(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
