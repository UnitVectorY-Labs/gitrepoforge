package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

// TemplateData is the data passed to templates for rendering.
type TemplateData struct {
	Name   string
	Inputs map[string]interface{}
}

// ComputeFindings computes the compliance findings for a repo.
// It returns the list of findings (differences between desired and actual state).
func ComputeFindings(repoCfg *config.RepoConfig, centralCfg *config.CentralConfig, repoPath string) ([]Finding, error) {
	var findings []Finding

	data := TemplateData{
		Name:   repoCfg.Name,
		Inputs: repoCfg.Inputs,
	}

	for _, rule := range centralCfg.Files {
		if !evaluateCondition(rule.Condition, data) {
			continue
		}

		ruleFinding, err := evaluateFileRule(rule, data, repoPath)
		if err != nil {
			return nil, fmt.Errorf("error evaluating rule for %s: %w", rule.Path, err)
		}
		findings = append(findings, ruleFinding...)
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
		case "block_replace":
			if err := os.WriteFile(filePath, []byte(f.Expected), 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", filePath, err)
			}
		}
	}
	return nil
}

func evaluateFileRule(rule config.FileRule, data TemplateData, repoPath string) ([]Finding, error) {
	switch rule.Mode {
	case "create":
		return evaluateCreateRule(rule, data, repoPath)
	case "delete":
		return evaluateDeleteRule(rule, repoPath)
	case "partial":
		return evaluatePartialRule(rule, data, repoPath)
	default:
		return evaluateCreateRule(rule, data, repoPath)
	}
}

func evaluateCreateRule(rule config.FileRule, data TemplateData, repoPath string) ([]Finding, error) {
	if rule.Template == "" {
		return nil, fmt.Errorf("output rule for %s has no template", rule.Path)
	}

	expected, err := renderTemplateString(rule.Template, data)
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
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "delete",
			Message:   "file exists but should not",
		}}, nil
	}
	return nil, nil
}

func evaluatePartialRule(rule config.FileRule, data TemplateData, repoPath string) ([]Finding, error) {
	filePath := filepath.Join(repoPath, rule.Path)
	actual, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// For partial rules, if the file doesn't exist, we create it with just the managed blocks
			content, err := renderManagedBlocks(rule.Blocks, data)
			if err != nil {
				return nil, err
			}
			return []Finding{{
				FilePath:  rule.Path,
				Operation: "create",
				Message:   "file does not exist; managed blocks need to be created",
				Expected:  content,
			}}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	currentContent := string(actual)
	expectedContent := currentContent

	for _, block := range rule.Blocks {
		blockContent, err := renderBlockContent(block, data)
		if err != nil {
			return nil, err
		}
		expectedContent = replaceBlock(expectedContent, block.BeginMarker, block.EndMarker, blockContent)
	}

	if currentContent != expectedContent {
		return []Finding{{
			FilePath:  rule.Path,
			Operation: "block_replace",
			Message:   "managed blocks differ from expected",
			Expected:  expectedContent,
			Actual:    currentContent,
		}}, nil
	}

	return nil, nil
}

func renderTemplateString(tmplStr string, data TemplateData) (string, error) {
	funcMap := template.FuncMap{
		"contains": func(list []interface{}, item string) bool {
			for _, v := range list {
				if fmt.Sprintf("%v", v) == item {
					return true
				}
			}
			return false
		},
		"join": func(list []interface{}, sep string) string {
			var strs []string
			for _, v := range list {
				strs = append(strs, fmt.Sprintf("%v", v))
			}
			return strings.Join(strs, sep)
		},
		"getInput": func(inputs map[string]interface{}, key string) interface{} {
			if inputs == nil {
				return nil
			}
			return inputs[key]
		},
	}

	tmpl, err := template.New("content").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

func renderManagedBlocks(blocks []config.BlockRule, data TemplateData) (string, error) {
	var parts []string
	for _, block := range blocks {
		content, err := renderBlockContent(block, data)
		if err != nil {
			return "", err
		}
		parts = append(parts, block.BeginMarker+"\n"+content+"\n"+block.EndMarker)
	}
	return strings.Join(parts, "\n"), nil
}

func renderBlockContent(block config.BlockRule, data TemplateData) (string, error) {
	if block.Template != "" {
		return renderTemplateString(block.Template, data)
	}
	return "", nil
}

func replaceBlock(content, beginMarker, endMarker, newBlock string) string {
	beginIdx := strings.Index(content, beginMarker)
	endIdx := strings.Index(content, endMarker)

	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		// Markers not found: append the block at the end
		return content + "\n" + beginMarker + "\n" + newBlock + "\n" + endMarker + "\n"
	}

	// Replace content between markers (inclusive of markers)
	return content[:beginIdx] + beginMarker + "\n" + newBlock + "\n" + endMarker + content[endIdx+len(endMarker):]
}

func evaluateCondition(condition string, data TemplateData) bool {
	if condition == "" {
		return true
	}

	// Simple condition evaluation: "input_name == value"
	parts := strings.SplitN(condition, "==", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		expected = strings.Trim(expected, "\"'")

		if data.Inputs != nil {
			if val, ok := data.Inputs[key]; ok {
				return fmt.Sprintf("%v", val) == expected
			}
		}
		return false
	}

	// Simple condition: "input_name != value"
	parts = strings.SplitN(condition, "!=", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		expected := strings.TrimSpace(parts[1])
		expected = strings.Trim(expected, "\"'")

		if data.Inputs != nil {
			if val, ok := data.Inputs[key]; ok {
				return fmt.Sprintf("%v", val) != expected
			}
		}
		return true
	}

	// Boolean input check: "input_name"
	if data.Inputs != nil {
		if val, ok := data.Inputs[condition]; ok {
			switch v := val.(type) {
			case bool:
				return v
			case string:
				return v != "" && v != "false"
			default:
				return true
			}
		}
	}
	return false
}
