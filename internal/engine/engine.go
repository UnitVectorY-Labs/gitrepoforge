package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

// Finding represents a single compliance finding for a file.
type Finding struct {
	FilePath  string `json:"file_path"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
}

// TemplateData is the data passed to template files for rendering.
type TemplateData struct {
	Name           string
	DefaultBranch  string
	Config         map[string]interface{}
	captures       map[string]map[string]string
	providedConfig map[string]interface{}
}

// ComputeFindings computes the compliance findings for a repo.
func ComputeFindings(repoCfg *config.RepoConfig, centralCfg *config.CentralConfig, repoPath, manifestPath string) ([]Finding, error) {
	var findings []Finding

	providedConfig := cloneConfigMap(repoCfg.Config)
	config.ApplyConfigDefaults(repoCfg, centralCfg)
	repoCfg.Config = config.ResolvedConfigValues(repoCfg, centralCfg)

	data := TemplateData{
		Name:           repoCfg.Name,
		DefaultBranch:  repoCfg.DefaultBranch,
		Config:         repoCfg.Config,
		captures:       computeCaptures(repoCfg.Config, centralCfg.Definitions),
		providedConfig: providedConfig,
	}

	for _, rule := range centralCfg.Files {
		ruleFindings, err := evaluateFileRule(rule, data, repoPath)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule for %s: %w", rule.Path, err)
		}
		findings = append(findings, ruleFindings...)
	}

	manifestContent, err := renderManagedFilesManifest(data, centralCfg, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("error rendering managed files manifest: %w", err)
	}
	manifestFindings, err := evaluateWholeFileRule(config.FileRule{Path: manifestPath}, manifestContent, repoPath)
	if err != nil {
		return nil, fmt.Errorf("error evaluating managed files manifest: %w", err)
	}
	findings = append(findings, manifestFindings...)

	return findings, nil
}

// ApplyFindings applies the given findings to the repository filesystem.
func ApplyFindings(findings []Finding, repoPath string) error {
	for _, f := range findings {
		filePath := filepath.Join(repoPath, f.FilePath)
		switch f.Operation {
		case "create", "update":
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(filePath, []byte(f.Expected), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", filePath, err)
			}
		case "delete":
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete file %s: %w", filePath, err)
			}
		}
	}
	return nil
}

func evaluateFileRule(rule config.FileRule, data TemplateData, repoPath string) ([]Finding, error) {
	switch rule.Mode {
	case "delete":
		return evaluateDeleteRule(rule, repoPath)
	case "create", "":
		return evaluateCreateRule(rule, data, repoPath)
	default:
		return nil, fmt.Errorf("unsupported mode %q", rule.Mode)
	}
}

func evaluateCreateRule(rule config.FileRule, data TemplateData, repoPath string) ([]Finding, error) {
	selected, err := selectTemplate(rule, data)
	if err != nil {
		return nil, err
	}

	if selected.Absent {
		return evaluateDeleteRule(rule, repoPath)
	}

	// Read raw template content to detect section directives before
	// Go template evaluation (directives use {{ }} syntax and must be
	// extracted first).
	rawContent, err := os.ReadFile(selected.ResolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", selected.ResolvedPath, err)
	}

	parsed, err := parseTemplateDirectives(string(rawContent))
	if err != nil {
		return nil, fmt.Errorf("parsing template directives for %s: %w", rule.Path, err)
	}

	if parsed.IsWholeFile {
		materialized, err := materializeTemplateFile(selected, data)
		if err != nil {
			return nil, err
		}
		return evaluateWholeFileRule(rule, materialized, repoPath)
	}

	// Section-based: evaluate content blocks if template evaluation is enabled
	if selected.Evaluate {
		for i := range parsed.Sections {
			rendered, err := renderTemplateContent(
				selected.ResolvedPath, parsed.Sections[i].Content,
				selected.TemplateMode, data,
			)
			if err != nil {
				return nil, fmt.Errorf("rendering section content for %s: %w", rule.Path, err)
			}
			parsed.Sections[i].Content = rendered
		}
		if parsed.HasBootstrap {
			rendered, err := renderTemplateContent(
				selected.ResolvedPath, parsed.BootstrapContent,
				selected.TemplateMode, data,
			)
			if err != nil {
				return nil, fmt.Errorf("rendering bootstrap content for %s: %w", rule.Path, err)
			}
			parsed.BootstrapContent = rendered
		}
	}

	return evaluateSectionRule(rule, parsed, repoPath)
}

func evaluateWholeFileRule(rule config.FileRule, expected string, repoPath string) ([]Finding, error) {
	filePath := filepath.Join(repoPath, rule.Path)
	actual, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Finding{{
				FilePath:  rule.Path,
				Operation: "create",
				Message:   "file does not exist but should",
				Expected:  expected,
			}}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	if string(actual) != expected {
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "update",
			Message:   "file content differs from expected",
			Expected:  expected,
			Actual:    string(actual),
		}}, nil
	}

	return nil, nil
}

func evaluateSectionRule(rule config.FileRule, parsed *ParsedTemplate, repoPath string) ([]Finding, error) {
	filePath := filepath.Join(repoPath, rule.Path)
	actual, err := os.ReadFile(filePath)
	fileExists := true
	if err != nil {
		if os.IsNotExist(err) {
			fileExists = false
		} else {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}
	}

	expected, err := applySections(parsed, string(actual), fileExists)
	if err != nil {
		return nil, fmt.Errorf("applying sections to %s: %w", rule.Path, err)
	}

	if !fileExists {
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "create",
			Message:   "file does not exist but should",
			Expected:  expected,
		}}, nil
	}

	if string(actual) != expected {
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "update",
			Message:   "file content differs from expected",
			Expected:  expected,
			Actual:    string(actual),
		}}, nil
	}

	return nil, nil
}

func evaluateDeleteRule(rule config.FileRule, repoPath string) ([]Finding, error) {
	filePath := filepath.Join(repoPath, rule.Path)
	if _, err := os.Stat(filePath); err == nil {
		actual, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, readErr)
		}
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "delete",
			Message:   "file exists but should not",
			Actual:    string(actual),
		}}, nil
	}
	return nil, nil
}

func selectTemplate(rule config.FileRule, data TemplateData) (config.TemplateRef, error) {
	if len(rule.Templates) == 0 {
		return config.TemplateRef{}, fmt.Errorf("output rule has no templates")
	}

	for _, candidate := range rule.Templates {
		matches, err := EvaluateCondition(candidate.Condition, data.Config, data.providedConfig)
		if err != nil {
			return config.TemplateRef{}, fmt.Errorf("invalid condition %q: %w", candidate.Condition, err)
		}
		if matches {
			return candidate, nil
		}
	}

	return config.TemplateRef{}, fmt.Errorf("no template matched")
}

func materializeTemplateFile(selected config.TemplateRef, data TemplateData) (string, error) {
	if !selected.Evaluate {
		content, err := os.ReadFile(selected.ResolvedPath)
		if err != nil {
			return "", fmt.Errorf("failed to read template %s: %w", selected.ResolvedPath, err)
		}
		return string(content), nil
	}

	return renderTemplateFile(selected.ResolvedPath, selected.TemplateMode, data)
}

func renderTemplateFile(path, templateMode string, data TemplateData) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", path, err)
	}
	preparedContent, placeholder := prepareTemplateContent(string(content), templateMode)

	funcMap := template.FuncMap{
		"getConfig": func(values map[string]interface{}, key string) interface{} {
			if values == nil {
				return nil
			}
			return values[key]
		},
		"quote_double": func(value interface{}) string {
			return quoteDoubleTemplateValue(value)
		},
		"quote_single": func(value interface{}) string {
			return quoteSingleTemplateValue(value)
		},
		"capture": func(key, group string) (string, error) {
			return lookupCapture(data.captures, key, group)
		},
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).Parse(preparedContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", path, err)
	}

	rendered := buf.String()
	if placeholder != "" {
		rendered = strings.ReplaceAll(rendered, placeholder, "{{")
	}

	return rendered, nil
}

// renderTemplateContent evaluates a content string as a Go template.
// Used for section and bootstrap content blocks that need template evaluation.
func renderTemplateContent(path, content, templateMode string, data TemplateData) (string, error) {
	preparedContent, placeholder := prepareTemplateContent(content, templateMode)

	funcMap := template.FuncMap{
		"getConfig": func(values map[string]interface{}, key string) interface{} {
			if values == nil {
				return nil
			}
			return values[key]
		},
		"quote_double": func(value interface{}) string {
			return quoteDoubleTemplateValue(value)
		},
		"quote_single": func(value interface{}) string {
			return quoteSingleTemplateValue(value)
		},
		"capture": func(key, group string) (string, error) {
			return lookupCapture(data.captures, key, group)
		},
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).Parse(preparedContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template content from %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template content from %s: %w", path, err)
	}

	rendered := buf.String()
	if placeholder != "" {
		rendered = strings.ReplaceAll(rendered, placeholder, "{{")
	}

	return rendered, nil
}

func prepareTemplateContent(content, templateMode string) (string, string) {
	switch config.TemplateModeOrDefault(templateMode) {
	case config.TemplateModeDoubleBracketStrict:
		placeholder := uniqueStrictModePlaceholder(content)
		return escapeStrictModeDoubleBrackets(content, placeholder), placeholder
	default:
		return content, ""
	}
}

func uniqueStrictModePlaceholder(content string) string {
	placeholder := "__GITREPOFORGE_ESCAPED_DOUBLE_BRACKET__"
	for strings.Contains(content, placeholder) {
		placeholder += "_"
	}
	return placeholder
}

func escapeStrictModeDoubleBrackets(content, placeholder string) string {
	if !strings.Contains(content, "{{") {
		return content
	}

	var builder strings.Builder
	builder.Grow(len(content))

	offset := 0
	for {
		index := strings.Index(content[offset:], "{{")
		if index == -1 {
			builder.WriteString(content[offset:])
			return builder.String()
		}

		index += offset
		builder.WriteString(content[offset:index])
		if hasStrictTemplateBoundary(content, index) {
			builder.WriteString("{{")
		} else {
			builder.WriteString(placeholder)
		}
		offset = index + len("{{")
	}
}

func quoteDoubleTemplateValue(value interface{}) string {
	return strconv.Quote(fmt.Sprint(value))
}

func quoteSingleTemplateValue(value interface{}) string {
	return "'" + strings.ReplaceAll(fmt.Sprint(value), "'", "''") + "'"
}

func hasStrictTemplateBoundary(content string, index int) bool {
	if index == 0 {
		return true
	}

	prev, _ := utf8.DecodeLastRuneInString(content[:index])
	return unicode.IsSpace(prev)
}

// EvaluateCondition checks whether a template selector condition matches the
// current repo config. Empty conditions always match.
func EvaluateCondition(condition string, values, providedValues map[string]interface{}) (bool, error) {
	parser := conditionParser{
		input:          strings.TrimSpace(condition),
		values:         values,
		providedValues: providedValues,
	}
	if parser.input == "" {
		return true, nil
	}

	result, err := parser.parseOr()
	if err != nil {
		return false, err
	}
	parser.skipWhitespace()
	if !parser.done() {
		return false, fmt.Errorf("unexpected token near %q", parser.input[parser.pos:])
	}
	return result, nil
}

type conditionParser struct {
	input          string
	pos            int
	values         map[string]interface{}
	providedValues map[string]interface{}
}

func (p *conditionParser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}

	for {
		p.skipWhitespace()
		if !p.match("||") {
			return left, nil
		}

		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}
}

func (p *conditionParser) parseAnd() (bool, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return false, err
	}

	for {
		p.skipWhitespace()
		if !p.match("&&") {
			return left, nil
		}

		right, err := p.parsePrimary()
		if err != nil {
			return false, err
		}
		left = left && right
	}
}

func (p *conditionParser) parsePrimary() (bool, error) {
	p.skipWhitespace()
	if p.done() {
		return false, fmt.Errorf("unexpected end of condition")
	}

	if p.match("(") {
		result, err := p.parseOr()
		if err != nil {
			return false, err
		}
		p.skipWhitespace()
		if !p.match(")") {
			return false, fmt.Errorf("missing closing parenthesis")
		}
		return result, nil
	}

	atom := p.readAtom()
	if atom == "" {
		return false, fmt.Errorf("unexpected token near %q", p.input[p.pos:])
	}
	return evaluateSimpleCondition(atom, p.values, p.providedValues)
}

func (p *conditionParser) readAtom() string {
	start := p.pos
	var quote byte
	for !p.done() {
		ch := p.input[p.pos]
		if quote != 0 {
			p.pos++
			if ch == '\\' && !p.done() {
				p.pos++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		switch ch {
		case '"', '\'':
			quote = ch
			p.pos++
		case '(':
			if p.pos == start {
				return ""
			}
			return strings.TrimSpace(p.input[start:p.pos])
		case ')':
			return strings.TrimSpace(p.input[start:p.pos])
		case '&':
			if strings.HasPrefix(p.input[p.pos:], "&&") {
				return strings.TrimSpace(p.input[start:p.pos])
			}
			p.pos++
		case '|':
			if strings.HasPrefix(p.input[p.pos:], "||") {
				return strings.TrimSpace(p.input[start:p.pos])
			}
			p.pos++
		default:
			p.pos++
		}
	}
	return strings.TrimSpace(p.input[start:p.pos])
}

func (p *conditionParser) skipWhitespace() {
	for !p.done() {
		switch p.input[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *conditionParser) match(token string) bool {
	if strings.HasPrefix(p.input[p.pos:], token) {
		p.pos += len(token)
		return true
	}
	return false
}

func (p *conditionParser) done() bool {
	return p.pos >= len(p.input)
}

func evaluateSimpleCondition(condition string, values, providedValues map[string]interface{}) (bool, error) {
	if key, negate, ok, err := parseExistsCondition(condition); ok {
		if err != nil {
			return false, err
		}
		_, exists := lookupConfigValue(key, providedValues)
		if negate {
			return !exists, nil
		}
		return exists, nil
	}

	if strings.HasPrefix(condition, "!") {
		result, err := evaluateBooleanKey(strings.TrimSpace(condition[1:]), values)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	if strings.Contains(condition, "!=") {
		parts := strings.SplitN(condition, "!=", 2)
		key := strings.TrimSpace(parts[0])
		if !isValidConditionKey(key) {
			return false, fmt.Errorf("invalid condition key %q", key)
		}
		expected := parseConditionValue(parts[1])
		actual, ok := lookupConfigValue(key, values)
		if !ok {
			return true, nil
		}
		return fmt.Sprintf("%v", actual) != expected, nil
	}

	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		key := strings.TrimSpace(parts[0])
		if !isValidConditionKey(key) {
			return false, fmt.Errorf("invalid condition key %q", key)
		}
		expected := parseConditionValue(parts[1])
		actual, ok := lookupConfigValue(key, values)
		if !ok {
			return false, nil
		}
		return fmt.Sprintf("%v", actual) == expected, nil
	}

	return evaluateBooleanKey(condition, values)
}

func parseExistsCondition(condition string) (key string, negate, ok bool, err error) {
	switch {
	case condition == "exists", condition == "!exists":
		return "", false, true, fmt.Errorf("invalid exists condition %q", condition)
	case strings.HasPrefix(condition, "exists "):
		key = strings.TrimSpace(condition[len("exists "):])
	case strings.HasPrefix(condition, "!exists "):
		key = strings.TrimSpace(condition[len("!exists "):])
		negate = true
	default:
		return "", false, false, nil
	}

	if !isValidConditionKey(key) {
		return "", false, true, fmt.Errorf("invalid exists condition %q", condition)
	}

	return key, negate, true, nil
}

func evaluateBooleanKey(key string, values map[string]interface{}) (bool, error) {
	key = strings.TrimSpace(key)
	if !isValidConditionKey(key) {
		return false, fmt.Errorf("invalid boolean condition %q", key)
	}

	actual, ok := lookupConfigValue(key, values)
	if !ok {
		return false, nil
	}

	boolean, ok := actual.(bool)
	if !ok {
		return false, fmt.Errorf("bare condition %q requires a boolean config value", key)
	}

	return boolean, nil
}

func lookupConfigValue(key string, values map[string]interface{}) (interface{}, bool) {
	if values == nil {
		return nil, false
	}
	current := interface{}(values)
	for _, part := range strings.Split(key, ".") {
		nested, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		value, exists := nested[part]
		if !exists {
			return nil, false
		}
		current = value
	}
	return current, true
}

func parseConditionValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

var conditionKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func isValidConditionKey(key string) bool {
	return conditionKeyPattern.MatchString(key)
}

func cloneConfigMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}

	cloned, ok := cloneConfigValue(values).(map[string]interface{})
	if !ok {
		return nil
	}
	return cloned
}

func cloneConfigValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, nestedValue := range typed {
			result[key] = cloneConfigValue(nestedValue)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, nestedValue := range typed {
			keyName, ok := key.(string)
			if !ok {
				return value
			}
			result[keyName] = cloneConfigValue(nestedValue)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for i := range typed {
			result[i] = cloneConfigValue(typed[i])
		}
		return result
	default:
		return value
	}
}

// computeCaptures extracts named capture groups from config values that have
// patterns defined. The result maps dotted key paths to group-name/value pairs.
func computeCaptures(values map[string]interface{}, definitions []config.ConfigDefinition) map[string]map[string]string {
	captures := map[string]map[string]string{}
	for _, def := range definitions {
		if def.Type == "object" {
			nested, ok := config.AsConfigMap(values[def.Name])
			if !ok {
				continue
			}
			for k, v := range computeCaptures(nested, def.Attributes) {
				captures[def.Name+"."+k] = v
			}
			continue
		}
		if def.CompiledPattern == nil {
			continue
		}
		val, ok := values[def.Name].(string)
		if !ok {
			continue
		}
		match := def.CompiledPattern.FindStringSubmatch(val)
		if match == nil {
			continue
		}
		groups := map[string]string{}
		for i, name := range def.CompiledPattern.SubexpNames() {
			if name != "" {
				groups[name] = match[i]
			}
		}
		captures[def.Name] = groups
	}
	return captures
}

// lookupCapture retrieves a named capture group value from the precomputed
// captures map. Returns an error when no pattern or group is found.
func lookupCapture(captures map[string]map[string]string, key, group string) (string, error) {
	if captures == nil {
		return "", fmt.Errorf("no pattern defined for config key %q", key)
	}
	keyCaptures, ok := captures[key]
	if !ok {
		return "", fmt.Errorf("no pattern defined for config key %q", key)
	}
	val, ok := keyCaptures[group]
	if !ok {
		return "", fmt.Errorf("no capture group %q defined in pattern for config key %q", group, key)
	}
	return val, nil
}
