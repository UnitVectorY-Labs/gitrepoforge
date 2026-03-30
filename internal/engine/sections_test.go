package engine

import (
	"testing"
)

func TestProcessJoinBlocks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "no join blocks",
			input: "line1\nline2\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name:  "simple join",
			input: "#!gitrepoforge:join\na\nb\nc\n#!gitrepoforge:end",
			want:  "abc",
		},
		{
			name:  "join within other content",
			input: "before\n#!gitrepoforge:join\na\nb\n#!gitrepoforge:end\nafter",
			want:  "before\nab\nafter",
		},
		{
			name:  "join with empty lines skipped",
			input: "#!gitrepoforge:join\na\n\nb\n#!gitrepoforge:end",
			want:  "ab",
		},
		{
			name:    "unterminated join",
			input:   "#!gitrepoforge:join\na\nb",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processJoinBlocks(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("processJoinBlocks() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("processJoinBlocks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBoundary(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Boundary
		wantErr bool
	}{
		{
			name:  "start_of_file",
			input: "start_of_file",
			want:  Boundary{Type: boundaryStartOfFile},
		},
		{
			name:  "end_of_file",
			input: "end_of_file",
			want:  Boundary{Type: boundaryEndOfFile},
		},
		{
			name:  "line number",
			input: `line("5")`,
			want:  Boundary{Type: boundaryLine, Value: "5"},
		},
		{
			name:  "content match",
			input: `content("<!-- END -->")`,
			want:  Boundary{Type: boundaryContent, Value: "<!-- END -->"},
		},
		{
			name:  "contains match",
			input: `contains("FOOTER")`,
			want:  Boundary{Type: boundaryContains, Value: "FOOTER"},
		},
		{
			name:    "invalid line number",
			input:   `line("abc")`,
			wantErr: true,
		},
		{
			name:    "unknown boundary",
			input:   "foobar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBoundary(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseBoundary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Type != tt.want.Type || got.Value != tt.want.Value {
					t.Fatalf("parseBoundary() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestParseTemplateDirectives(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantWhole  bool
		wantSecs   int
		wantBoot   bool
		wantErr    bool
	}{
		{
			name:      "no directives returns whole file",
			input:     "just regular content\nline two",
			wantWhole: true,
		},
		{
			name: "single section",
			input: `#!gitrepoforge:section start=start_of_file end=contains("<!-- END -->")
# Header
<!-- END -->
#!gitrepoforge:end`,
			wantSecs: 1,
		},
		{
			name: "section with bootstrap",
			input: `#!gitrepoforge:section start=start_of_file end=contains("<!-- END -->")
# Header
<!-- END -->
#!gitrepoforge:end
#!gitrepoforge:bootstrap
Default body
#!gitrepoforge:end`,
			wantSecs: 1,
			wantBoot: true,
		},
		{
			name: "multiple sections",
			input: `#!gitrepoforge:section start=start_of_file end=content("<!-- DIVIDER -->")
Header
<!-- DIVIDER -->
#!gitrepoforge:end
#!gitrepoforge:section start=contains("<!-- FOOTER -->") end=end_of_file
<!-- FOOTER -->
Footer
#!gitrepoforge:end`,
			wantSecs: 2,
		},
		{
			name: "bootstrap only",
			input: `#!gitrepoforge:bootstrap
some content
#!gitrepoforge:end`,
			wantBoot: true,
		},
		{
			name:    "content outside sections is error",
			input:   "some content\n#!gitrepoforge:section start=start_of_file end=end_of_file\ncontent\n#!gitrepoforge:end",
			wantErr: true,
		},
		{
			name: "empty bootstrap",
			input: `#!gitrepoforge:bootstrap
#!gitrepoforge:end`,
			wantBoot: true,
		},
		{
			name:    "unterminated section",
			input:   "#!gitrepoforge:section start=start_of_file end=end_of_file\ncontent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTemplateDirectives(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseTemplateDirectives() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.IsWholeFile != tt.wantWhole {
				t.Fatalf("IsWholeFile = %v, want %v", got.IsWholeFile, tt.wantWhole)
			}
			if len(got.Sections) != tt.wantSecs {
				t.Fatalf("len(Sections) = %d, want %d", len(got.Sections), tt.wantSecs)
			}
			if got.HasBootstrap != tt.wantBoot {
				t.Fatalf("HasBootstrap = %v, want %v", got.HasBootstrap, tt.wantBoot)
			}
		})
	}
}

func TestResolveBoundary(t *testing.T) {
	lines := []string{"line one", "line two", "line three", "line four", "line five"}

	tests := []struct {
		name       string
		boundary   Boundary
		searchFrom int
		wantIdx    int
		wantErr    bool
	}{
		{
			name:     "start_of_file",
			boundary: Boundary{Type: boundaryStartOfFile},
			wantIdx:  0,
		},
		{
			name:     "end_of_file",
			boundary: Boundary{Type: boundaryEndOfFile},
			wantIdx:  4,
		},
		{
			name:     "line number",
			boundary: Boundary{Type: boundaryLine, Value: "3"},
			wantIdx:  2,
		},
		{
			name:     "content match",
			boundary: Boundary{Type: boundaryContent, Value: "line three"},
			wantIdx:  2,
		},
		{
			name:     "contains match",
			boundary: Boundary{Type: boundaryContains, Value: "four"},
			wantIdx:  3,
		},
		{
			name:       "content not found",
			boundary:   Boundary{Type: boundaryContent, Value: "missing"},
			wantErr:    true,
		},
		{
			name:       "contains not found",
			boundary:   Boundary{Type: boundaryContains, Value: "missing"},
			wantErr:    true,
		},
		{
			name:       "line out of range",
			boundary:   Boundary{Type: boundaryLine, Value: "10"},
			wantErr:    true,
		},
		{
			name:       "contains with search from",
			boundary:   Boundary{Type: boundaryContains, Value: "line"},
			searchFrom: 2,
			wantIdx:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBoundary(tt.boundary, lines, tt.searchFrom)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveBoundary() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.wantIdx {
				t.Fatalf("resolveBoundary() = %d, want %d", got, tt.wantIdx)
			}
		})
	}
}

func TestApplySections(t *testing.T) {
	tests := []struct {
		name       string
		parsed     *ParsedTemplate
		content    string
		exists     bool
		want       string
		wantErr    bool
	}{
		{
			name: "new file with section",
			parsed: &ParsedTemplate{
				Sections: []Section{
					{
						Start:   Boundary{Type: boundaryStartOfFile},
						End:     Boundary{Type: boundaryContains, Value: "<!-- END -->"},
						Content: "# Title\n<!-- END -->",
					},
				},
			},
			exists: false,
			want:   "# Title\n<!-- END -->\n",
		},
		{
			name: "new file with section and bootstrap",
			parsed: &ParsedTemplate{
				Sections: []Section{
					{
						Start:   Boundary{Type: boundaryStartOfFile},
						End:     Boundary{Type: boundaryContains, Value: "<!-- END -->"},
						Content: "# Title\n<!-- END -->",
					},
				},
				HasBootstrap:     true,
				BootstrapContent: "Default body.",
			},
			exists: false,
			want:   "# Title\n<!-- END -->\nDefault body.\n",
		},
		{
			name: "existing file with header section",
			parsed: &ParsedTemplate{
				Sections: []Section{
					{
						Start:   Boundary{Type: boundaryStartOfFile},
						End:     Boundary{Type: boundaryContains, Value: "<!-- END -->"},
						Content: "# New Title\n<!-- END -->",
					},
				},
			},
			content: "# Old Title\n<!-- END -->\nUser content\n",
			exists:  true,
			want:    "# New Title\n<!-- END -->\nUser content\n",
		},
		{
			name: "bootstrap only new file empty",
			parsed: &ParsedTemplate{
				HasBootstrap:     true,
				BootstrapContent: "",
			},
			exists: false,
			want:   "",
		},
		{
			name: "bootstrap only existing file unchanged",
			parsed: &ParsedTemplate{
				HasBootstrap:     true,
				BootstrapContent: "",
			},
			content: "existing content\n",
			exists:  true,
			want:    "existing content\n",
		},
		{
			name: "boundary not found in existing file",
			parsed: &ParsedTemplate{
				Sections: []Section{
					{
						Start:   Boundary{Type: boundaryStartOfFile},
						End:     Boundary{Type: boundaryContains, Value: "<!-- MISSING -->"},
						Content: "content",
					},
				},
			},
			content: "file without marker\n",
			exists:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applySections(tt.parsed, tt.content, tt.exists)
			if (err != nil) != tt.wantErr {
				t.Fatalf("applySections() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("applySections() = %q, want %q", got, tt.want)
			}
		})
	}
}
