package engine

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	directivePrefix = "#!gitrepoforge:"
	directiveEnd    = "#!gitrepoforge:end"

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
// materialized template. If no directives are found, IsWholeFile is true
// and the template should replace the entire file (backward compatible).
type ParsedTemplate struct {
	Sections          []Section
	BootstrapContent  string
	HasBootstrap      bool
	IsWholeFile       bool
	OriginalContent   string
}

// parseTemplateDirectives parses the materialized template content for section
// directives. If no directives are found, it returns a ParsedTemplate with
// IsWholeFile=true and OriginalContent set to the full content.
func parseTemplateDirectives(content string) (*ParsedTemplate, error) {
	// First, process join blocks
	processed, err := processJoinBlocks(content)
	if err != nil {
		return nil, err
	}

	// Check if there are any directives at all
	if !strings.Contains(processed, directivePrefix) {
		return &ParsedTemplate{
			IsWholeFile:     true,
			OriginalContent: content,
		}, nil
	}

	result := &ParsedTemplate{}

	lines := strings.Split(processed, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if line == "" || !strings.HasPrefix(line, directivePrefix) {
			// Content outside directives - must be blank
			if line != "" {
				return nil, fmt.Errorf("content outside of section directives is not allowed: %q", lines[i])
			}
			i++
			continue
		}

		directive := line[len(directivePrefix):]

		if strings.HasPrefix(directive, "section ") {
			section, endIdx, err := parseSectionDirective(directive[len("section "):], lines, i)
			if err != nil {
				return nil, err
			}
			result.Sections = append(result.Sections, section)
			i = endIdx + 1
			continue
		}

		if directive == "bootstrap" {
			content, endIdx, err := parseBlockContent(lines, i)
			if err != nil {
				return nil, fmt.Errorf("bootstrap block: %w", err)
			}
			result.BootstrapContent = content
			result.HasBootstrap = true
			i = endIdx + 1
			continue
		}

		return nil, fmt.Errorf("unknown directive: %q", line)
	}

	return result, nil
}

// parseSectionDirective parses a section directive's boundaries and content.
func parseSectionDirective(params string, lines []string, startLine int) (Section, int, error) {
	start, end, err := parseSectionBoundaries(params)
	if err != nil {
		return Section{}, 0, fmt.Errorf("section directive: %w", err)
	}

	content, endIdx, err := parseBlockContent(lines, startLine)
	if err != nil {
		return Section{}, 0, fmt.Errorf("section block: %w", err)
	}

	return Section{
		Start:   start,
		End:     end,
		Content: content,
	}, endIdx, nil
}

// parseSectionBoundaries parses the start= and end= parameters from a section
// directive line.
func parseSectionBoundaries(params string) (Boundary, Boundary, error) {
	params = strings.TrimSpace(params)

	startStr, endStr, err := extractBoundaryParams(params)
	if err != nil {
		return Boundary{}, Boundary{}, err
	}

	start, err := parseBoundary(startStr)
	if err != nil {
		return Boundary{}, Boundary{}, fmt.Errorf("start boundary: %w", err)
	}

	end, err := parseBoundary(endStr)
	if err != nil {
		return Boundary{}, Boundary{}, fmt.Errorf("end boundary: %w", err)
	}

	return start, end, nil
}

// extractBoundaryParams extracts start= and end= values from the params
// string. It handles quoted values with parentheses inside.
func extractBoundaryParams(params string) (string, string, error) {
	var startVal, endVal string

	// Parse key=value pairs, handling function-style values like content("text")
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

	if startVal == "" {
		return "", "", fmt.Errorf("missing start parameter")
	}
	if endVal == "" {
		return "", "", fmt.Errorf("missing end parameter")
	}

	return startVal, endVal, nil
}

// extractParamValue extracts a boundary value from the params string.
// It handles simple values like start_of_file and function-style like content("text").
func extractParamValue(s string) (string, string, error) {
	s = strings.TrimSpace(s)

	// Check for function-style: name("value")
	for _, funcName := range []string{boundaryLine, boundaryContent, boundaryContains} {
		prefix := funcName + "("
		if strings.HasPrefix(s, prefix) {
			// Find the closing quote and parenthesis
			rest := s[len(prefix):]
			if len(rest) < 3 || rest[0] != '"' {
				return "", "", fmt.Errorf("expected quoted argument in %s()", funcName)
			}
			// Find closing ")"
			endQuote := strings.Index(rest[1:], "\"")
			if endQuote == -1 {
				return "", "", fmt.Errorf("unterminated quoted argument in %s()", funcName)
			}
			endQuote++ // adjust for the offset
			if endQuote+1 >= len(rest) || rest[endQuote+1] != ')' {
				return "", "", fmt.Errorf("expected closing parenthesis in %s()", funcName)
			}
			value := s[:len(prefix)+endQuote+2] // include the closing )
			remaining := s[len(prefix)+endQuote+2:]
			return value, remaining, nil
		}
	}

	// Simple value (no quotes/parens): read until whitespace
	end := strings.IndexAny(s, " \t")
	if end == -1 {
		return s, "", nil
	}
	return s[:end], s[end:], nil
}

// parseBoundary parses a boundary specification string into a Boundary.
func parseBoundary(spec string) (Boundary, error) {
	spec = strings.TrimSpace(spec)

	switch spec {
	case boundaryStartOfFile:
		return Boundary{Type: boundaryStartOfFile}, nil
	case boundaryEndOfFile:
		return Boundary{Type: boundaryEndOfFile}, nil
	}

	// Check function-style boundaries
	for _, funcName := range []string{boundaryLine, boundaryContent, boundaryContains} {
		prefix := funcName + "(\""
		suffix := "\")"
		if strings.HasPrefix(spec, prefix) && strings.HasSuffix(spec, suffix) {
			value := spec[len(prefix) : len(spec)-len(suffix)]
			if funcName == boundaryLine {
				if _, err := strconv.Atoi(value); err != nil {
					return Boundary{}, fmt.Errorf("line boundary must be a number: %q", value)
				}
			}
			return Boundary{Type: funcName, Value: value}, nil
		}
	}

	return Boundary{}, fmt.Errorf("unknown boundary: %q", spec)
}

// parseBlockContent reads lines from startLine+1 until a #!gitrepoforge:end
// directive is found. Returns the content between the start and end directives
// and the index of the end directive line.
func parseBlockContent(lines []string, startLine int) (string, int, error) {
	var contentLines []string
	i := startLine + 1
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == directiveEnd {
			return strings.Join(contentLines, "\n"), i, nil
		}
		contentLines = append(contentLines, lines[i])
		i++
	}
	return "", 0, fmt.Errorf("unterminated block starting at line %d", startLine+1)
}

// processJoinBlocks finds all #!gitrepoforge:join ... #!gitrepoforge:end blocks
// and replaces them with the content joined into a single line (newlines removed).
func processJoinBlocks(content string) (string, error) {
	joinDirective := directivePrefix + "join"

	if !strings.Contains(content, joinDirective) {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	var result []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == joinDirective {
			// Collect lines until #!gitrepoforge:end
			i++
			var joinedParts []string
			found := false
			for i < len(lines) {
				trimmedInner := strings.TrimSpace(lines[i])
				if trimmedInner == directiveEnd {
					found = true
					break
				}
				// Remove \r and \n from lines and join them
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
// If the file does not exist (fileContent is empty string with fileExists=false),
// it creates the initial content from sections and bootstrap content.
func applySections(parsed *ParsedTemplate, fileContent string, fileExists bool) (string, error) {
	if !fileExists {
		return buildNewFileFromSections(parsed)
	}

	return applyToExistingFile(parsed, fileContent)
}

// buildNewFileFromSections creates initial file content from sections and
// bootstrap content. Sections are concatenated in order, with bootstrap
// content placed between sections (in the order it appears in the template).
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

	// Apply each section in order
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

		// Replace lines from startIdx to endIdx (inclusive) with section content
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

	// Preserve the original file's trailing newline convention
	if trailingNewline && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result, nil
}

// resolveBoundary finds the line index for a boundary in the given lines.
// searchFrom is used for end boundaries to start searching from the start
// boundary position.
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
		lineNum, _ := strconv.Atoi(b.Value) // already validated in parse
		idx := lineNum - 1                   // convert to 0-based
		if idx < 0 || idx >= len(lines) {
			return 0, fmt.Errorf("line number %d is out of range (file has %d lines)", lineNum, len(lines))
		}
		return idx, nil

	case boundaryContent:
		for i := searchFrom; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == b.Value {
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
