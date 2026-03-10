package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

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
	Name          string
	DefaultBranch string
	Config        map[string]interface{}
}

// ComputeFindings computes the compliance findings for a repo.
func ComputeFindings(repoCfg *config.RepoConfig, centralCfg *config.CentralConfig, repoPath string) ([]Finding, error) {
	var findings []Finding

	config.ApplyConfigDefaults(repoCfg, centralCfg)

	data := TemplateData{
		Name:          repoCfg.Name,
		DefaultBranch: repoCfg.DefaultBranch,
		Config:        repoCfg.Config,
	}

	for _, rule := range centralCfg.Files {
		ruleFindings, err := evaluateFileRule(rule, data, repoPath)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule for %s: %w", rule.Path, err)
		}
		findings = append(findings, ruleFindings...)
	}

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

	expected, err := materializeTemplateFile(selected, data)
	if err != nil {
		return nil, err
	}

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
		matches, err := EvaluateCondition(candidate.Condition, data.Config)
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

	return renderTemplateFile(selected.ResolvedPath, data)
}

func renderTemplateFile(path string, data TemplateData) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", path, err)
	}

	funcMap := template.FuncMap{
		"getConfig": func(values map[string]interface{}, key string) interface{} {
			if values == nil {
				return nil
			}
			return values[key]
		},
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", path, err)
	}

	return buf.String(), nil
}

// EvaluateCondition checks whether a template selector condition matches the
// current repo config. Empty conditions always match.
func EvaluateCondition(condition string, values map[string]interface{}) (bool, error) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true, nil
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
	value, ok := values[key]
	return value, ok
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
