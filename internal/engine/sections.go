package engine

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	boundaryStartOfFile = "start_of_file"
	boundaryEndOfFile   = "end_of_file"
	boundaryLine        = "line"
	boundaryContent     = "content"
	boundaryContains    = "contains"
)

// Boundary defines where a section starts or ends in a file.
type Boundary struct {
	Type  string // start_of_file, end_of_file, line, content, contains
	Value string // For line: number string; for content/contains: match text
}

// Section represents a managed section of a file defined in a template.
type Section struct {
	Start   Boundary
	End     Boundary
	Content string
}

// ParsedTemplate is the result of parsing section directives from a
// template. If no directives are found, IsWholeFile is true and the
// template should replace the entire file (backward compatible).
type ParsedTemplate struct {
	Sections         []Section
	BootstrapContent string
	HasBootstrap     bool
	IsWholeFile      bool
	OriginalContent  string
}

// parseTemplateDirectives parses the raw template content for section
// directives using {{ }} syntax. If no directives are found, it returns
// a ParsedTemplate with IsWholeFile=true.
func parseTemplateDirectives(content string) (*ParsedTemplate, error) {
	lines := strings.Split(content, "\n")

	if !hasStructuralDirectives(lines) {
		return &ParsedTemplate{
			IsWholeFile:     true,
			OriginalContent: content,
		}, nil
	}

	result := &ParsedTemplate{}
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if trimmed == "" {
			i++
			continue
		}

		inner := extractDirectiveInner(trimmed)
		if inner == "" {
			return nil, fmt.Errorf("content outside of section directives is not allowed: %q", lines[i])
		}

		keyword := directiveKeyword(inner)

		switch keyword {
		case "section":
			params := strings.TrimSpace(strings.TrimPrefix(inner, "section"))
			section, endIdx, err := parseSectionBlock(params, lines, i)
			if err != nil {
				return nil, err
			}
			result.Sections = append(result.Sections, section)
			i = endIdx + 1

		case "bootstrap":
			content, endIdx, err := collectBlockContent(lines, i)
			if err != nil {
				return nil, fmt.Errorf("bootstrap block: %w", err)
			}
			result.BootstrapContent = content
			result.HasBootstrap = true
			i = endIdx + 1

		default:
			return nil, fmt.Errorf("unexpected directive: %q", trimmed)
		}
	}

	return result, nil
}

// hasStructuralDirectives checks if any line in the template is a
// structural directive (section or bootstrap).
func hasStructuralDirectives(lines []string) bool {
	for _, line := range lines {
		inner := extractDirectiveInner(strings.TrimSpace(line))
		if inner == "" {
			continue
		}
		kw := directiveKeyword(inner)
		if kw == "section" || kw == "bootstrap" {
			return true
		}
	}
	return false
}

// extractDirectiveInner extracts the inner content of a {{ }} directive
// line with optional trim markers stripped. Returns empty string if the
// line is not a single standalone directive (does not start with {{ and
// end with }}, or contains embedded {{ / }} pairs).
func extractDirectiveInner(trimmed string) string {
	if !strings.HasPrefix(trimmed, "{{") || !strings.HasSuffix(trimmed, "}}") {
		return ""
	}
	inner := trimmed[2 : len(trimmed)-2]
	// If the inner text contains {{ or }}, this is not a standalone
	// directive but rather multiple Go template actions on one line.
	if strings.Contains(inner, "{{") || strings.Contains(inner, "}}") {
		return ""
	}
	inner = strings.TrimSpace(inner)
	if strings.HasPrefix(inner, "-") {
		inner = strings.TrimSpace(inner[1:])
	}
	if strings.HasSuffix(inner, "-") {
		inner = strings.TrimSpace(inner[:len(inner)-1])
	}
	return inner
}

// directiveKeyword returns the first word of the directive inner text.
func directiveKeyword(inner string) string {
	parts := strings.Fields(inner)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// parseSectionBlock parses a section directive and collects content until
// {{ end }}.
func parseSectionBlock(params string, lines []string, startLine int) (Section, int, error) {
	start, end, err := parseSectionBoundaries(params)
	if err != nil {
		return Section{}, 0, fmt.Errorf("section directive: %w", err)
	}

	content, endIdx, err := collectBlockContent(lines, startLine)
	if err != nil {
		return Section{}, 0, fmt.Errorf("section block: %w", err)
	}

	// Process join blocks within the section content
	processed, err := processJoinBlocks(content)
	if err != nil {
		return Section{}, 0, err
	}

	return Section{
		Start:   start,
		End:     end,
		Content: processed,
	}, endIdx, nil
}

// parseSectionBoundaries parses start= and end= parameters from a section
// directive. Both, one, or the other can be specified. Missing start
// defaults to start_of_file. Missing end defaults to end_of_file.
func parseSectionBoundaries(params string) (Boundary, Boundary, error) {
	params = strings.TrimSpace(params)
	if params == "" {
		return Boundary{}, Boundary{}, fmt.Errorf("section requires at least start or end parameter")
	}

	startStr, endStr, err := extractBoundaryParams(params)
	if err != nil {
		return Boundary{}, Boundary{}, err
	}

	var start Boundary
	if startStr != "" {
		start, err = parseBoundary(startStr)
		if err != nil {
			return Boundary{}, Boundary{}, fmt.Errorf("start boundary: %w", err)
		}
	} else {
		start = Boundary{Type: boundaryStartOfFile}
	}

	var endBoundary Boundary
	if endStr != "" {
		endBoundary, err = parseBoundary(endStr)
		if err != nil {
			return Boundary{}, Boundary{}, fmt.Errorf("end boundary: %w", err)
		}
	} else {
		endBoundary = Boundary{Type: boundaryEndOfFile}
	}

	return start, endBoundary, nil
}

// extractBoundaryParams extracts start= and end= values from a parameter
// string. Either or both can be present.
func extractBoundaryParams(params string) (string, string, error) {
	var startVal, endVal string

	remaining := params
	for remaining != "" {
		remaining = strings.TrimSpace(remaining)
		if remaining == "" {
			break
		}

		if strings.HasPrefix(remaining, "start=") {
			val, rest, err := extractParamValue(remaining[len("start="):])
			if err != nil {
				return "", "", fmt.Errorf("parsing start: %w", err)
			}
			startVal = val
			remaining = rest
		} else if strings.HasPrefix(remaining, "end=") {
			val, rest, err := extractParamValue(remaining[len("end="):])
			if err != nil {
				return "", "", fmt.Errorf("parsing end: %w", err)
			}
			endVal = val
			remaining = rest
		} else {
			return "", "", fmt.Errorf("unexpected parameter: %q", remaining)
		}
	}

	if startVal == "" && endVal == "" {
		return "", "", fmt.Errorf("section requires at least start or end parameter")
	}

	return startVal, endVal, nil
}

// extractParamValue extracts a boundary value from a parameter string.
// Handles simple values like start_of_file and function-style values
// like content("text") or line(3).
func extractParamValue(s string) (string, string, error) {
	s = strings.TrimSpace(s)

	for _, funcName := range []string{boundaryLine, boundaryContent, boundaryContains} {
		prefix := funcName + "("
		if !strings.HasPrefix(s, prefix) {
			continue
		}
		rest := s[len(prefix):]
		if len(rest) == 0 {
			return "", "", fmt.Errorf("empty argument in %s()", funcName)
		}
		if rest[0] == '"' {
			endQuote := strings.Index(rest[1:], "\"")
			if endQuote == -1 {
				return "", "", fmt.Errorf("unterminated quoted argument in %s()", funcName)
			}
			endQuote++ // adjust for offset
			if endQuote+1 >= len(rest) || rest[endQuote+1] != ')' {
				return "", "", fmt.Errorf("expected closing parenthesis in %s()", funcName)
			}
			value := s[:len(prefix)+endQuote+2]
			remaining := s[len(prefix)+endQuote+2:]
			return value, remaining, nil
		}
		closeParen := strings.Index(rest, ")")
		if closeParen == -1 {
			return "", "", fmt.Errorf("expected closing parenthesis in %s()", funcName)
		}
		value := s[:len(prefix)+closeParen+1]
		remaining := s[len(prefix)+closeParen+1:]
		return value, remaining, nil
	}

	end := strings.IndexAny(s, " \t")
	if end == -1 {
		return s, "", nil
	}
	return s[:end], s[end:], nil
}

// parseBoundary parses a boundary specification string into a Boundary.
// Supports start_of_file, end_of_file, line(N), content("text"), and
// contains("text"). Quoted values in line() are optional.
func parseBoundary(spec string) (Boundary, error) {
	spec = strings.TrimSpace(spec)

	switch spec {
	case boundaryStartOfFile:
		return Boundary{Type: boundaryStartOfFile}, nil
	case boundaryEndOfFile:
		return Boundary{Type: boundaryEndOfFile}, nil
	}

	for _, funcName := range []string{boundaryLine, boundaryContent, boundaryContains} {
		prefix := funcName + "("
		if !strings.HasPrefix(spec, prefix) || !strings.HasSuffix(spec, ")") {
			continue
		}
		if len(spec) <= len(prefix)+1 {
			return Boundary{}, fmt.Errorf("empty argument in %s()", funcName)
		}
		value := spec[len(prefix) : len(spec)-1]
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		if funcName == boundaryLine {
			if _, err := strconv.Atoi(value); err != nil {
				return Boundary{}, fmt.Errorf("line boundary must be a number: %q", value)
			}
		}
		return Boundary{Type: funcName, Value: value}, nil
	}

	return Boundary{}, fmt.Errorf("unknown boundary: %q", spec)
}

// isGoTemplateBlockOpener returns true if the keyword opens a Go
// template block that requires a matching {{ end }}.
func isGoTemplateBlockOpener(keyword string) bool {
	switch keyword {
	case "if", "range", "with", "block", "define":
		return true
	}
	return false
}

// collectBlockContent reads lines from startLine+1 until a closing
// {{ end }} directive is found at nesting depth 0. Go template block
// openers (if, range, with, block, define) and our own join blocks
// increment the nesting depth so that their {{ end }} directives are
// collected as content rather than treated as the block closer.
func collectBlockContent(lines []string, startLine int) (string, int, error) {
	var contentLines []string
	depth := 0
	i := startLine + 1
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		inner := extractDirectiveInner(trimmed)
		if inner != "" {
			kw := directiveKeyword(inner)
			if isGoTemplateBlockOpener(kw) || kw == "join" {
				depth++
			} else if kw == "end" {
				if depth == 0 {
					return strings.Join(contentLines, "\n"), i, nil
				}
				depth--
			}
		}
		contentLines = append(contentLines, lines[i])
		i++
	}
	return "", 0, fmt.Errorf("unterminated block starting at line %d: expected {{ end }}", startLine+1)
}

// processJoinBlocks finds {{ join }}...{{ end }} blocks within content
// and replaces them with the content joined into a single line.
func processJoinBlocks(content string) (string, error) {
	lines := strings.Split(content, "\n")

	hasJoin := false
	for _, line := range lines {
		inner := extractDirectiveInner(strings.TrimSpace(line))
		if inner != "" && directiveKeyword(inner) == "join" {
			hasJoin = true
			break
		}
	}
	if !hasJoin {
		return content, nil
	}

	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		inner := extractDirectiveInner(trimmed)
		if inner != "" && directiveKeyword(inner) == "join" {
			i++
			var joinedParts []string
			found := false
			depth := 0
			for i < len(lines) {
				trimmedInner := strings.TrimSpace(lines[i])
				innerJoin := extractDirectiveInner(trimmedInner)
				if innerJoin != "" {
					kw := directiveKeyword(innerJoin)
					if isGoTemplateBlockOpener(kw) {
						depth++
					} else if kw == "end" {
						if depth == 0 {
							found = true
							break
						}
						depth--
					}
				}
				// Strip trailing \r left over from \r\n line endings
				// since lines are already split on \n
				line := strings.TrimRight(lines[i], "\r")
				if line != "" {
					joinedParts = append(joinedParts, line)
				}
				i++
			}
			if !found {
				return "", fmt.Errorf("unterminated join block")
			}
			if len(joinedParts) > 0 {
				result = append(result, strings.Join(joinedParts, ""))
			}
			i++
			continue
		}
		result = append(result, lines[i])
		i++
	}

	return strings.Join(result, "\n"), nil
}

// applySections applies the parsed section directives to an existing file.
// If the file does not exist, it creates the initial content from sections
// and bootstrap content.
func applySections(parsed *ParsedTemplate, fileContent string, fileExists bool) (string, error) {
	if !fileExists {
		return buildNewFileFromSections(parsed)
	}

	return applyToExistingFile(parsed, fileContent)
}

// buildNewFileFromSections creates initial file content from sections and
// bootstrap content.
func buildNewFileFromSections(parsed *ParsedTemplate) (string, error) {
	var parts []string

	for _, section := range parsed.Sections {
		parts = append(parts, section.Content)
	}

	if parsed.HasBootstrap {
		parts = append(parts, parsed.BootstrapContent)
	}

	result := strings.Join(parts, "\n")

	// Ensure non-empty new files end with a newline, matching standard
	// text file conventions.
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result, nil
}

// applyToExistingFile applies managed sections to an existing file, preserving
// content outside the managed sections.
func applyToExistingFile(parsed *ParsedTemplate, fileContent string) (string, error) {
	trailingNewline := strings.HasSuffix(fileContent, "\n")
	lines := strings.Split(fileContent, "\n")

	for _, section := range parsed.Sections {
		startIdx, err := resolveBoundary(section.Start, lines, 0)
		if err != nil {
			return "", fmt.Errorf("resolving start boundary: %w", err)
		}

		searchFrom := startIdx
		endIdx, err := resolveBoundary(section.End, lines, searchFrom)
		if err != nil {
			return "", fmt.Errorf("resolving end boundary: %w", err)
		}

		sectionLines := strings.Split(section.Content, "\n")
		var newLines []string
		newLines = append(newLines, lines[:startIdx]...)
		newLines = append(newLines, sectionLines...)
		if endIdx+1 < len(lines) {
			newLines = append(newLines, lines[endIdx+1:]...)
		}
		lines = newLines
	}

	result := strings.Join(lines, "\n")

	if trailingNewline && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result, nil
}

// resolveBoundary finds the line index for a boundary in the given lines.
func resolveBoundary(b Boundary, lines []string, searchFrom int) (int, error) {
	switch b.Type {
	case boundaryStartOfFile:
		return 0, nil

	case boundaryEndOfFile:
		if len(lines) == 0 {
			return 0, nil
		}
		return len(lines) - 1, nil

	case boundaryLine:
		lineNum, _ := strconv.Atoi(b.Value)
		idx := lineNum - 1
		if idx < 0 || idx >= len(lines) {
			return 0, fmt.Errorf("line number %d is out of range (file has %d lines)", lineNum, len(lines))
		}
		return idx, nil

	case boundaryContent:
		for i := searchFrom; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == strings.TrimSpace(b.Value) {
				return i, nil
			}
		}
		return 0, fmt.Errorf("content boundary %q not found in file", b.Value)

	case boundaryContains:
		for i := searchFrom; i < len(lines); i++ {
			if strings.Contains(lines[i], b.Value) {
				return i, nil
			}
		}
		return 0, fmt.Errorf("contains boundary %q not found in file", b.Value)

	default:
		return 0, fmt.Errorf("unknown boundary type: %q", b.Type)
	}
}
