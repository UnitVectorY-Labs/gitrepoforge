package schema

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidateRepoConfig validates a repo config against the central config definitions.
// Returns a list of validation errors (empty if valid).
func ValidateRepoConfig(repoCfg *config.RepoConfig, centralCfg *config.CentralConfig, repoPath string) []ValidationError {
	var errors []ValidationError

	// Check that repo name matches folder name
	folderName := filepath.Base(repoPath)
	if repoCfg.Name == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "name is required"})
	} else if repoCfg.Name != folderName {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("name %q does not match folder name %q", repoCfg.Name, folderName),
		})
	}

	// Build allowed input names from central config
	allowedInputs := make(map[string]*config.InputDef)
	for i := range centralCfg.Inputs {
		allowedInputs[centralCfg.Inputs[i].Name] = &centralCfg.Inputs[i]
	}

	// Check for required inputs
	for _, inputDef := range centralCfg.Inputs {
		if inputDef.Required {
			if repoCfg.Inputs == nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("inputs.%s", inputDef.Name),
					Message: "required input is missing",
				})
			} else if _, ok := repoCfg.Inputs[inputDef.Name]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("inputs.%s", inputDef.Name),
					Message: "required input is missing",
				})
			}
		}
	}

	// Check for unknown inputs (strict no-extra-properties)
	if repoCfg.Inputs != nil {
		for key := range repoCfg.Inputs {
			if _, ok := allowedInputs[key]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("inputs.%s", key),
					Message: "unknown input",
				})
			}
		}
	}

	// Validate input values
	if repoCfg.Inputs != nil {
		for key, val := range repoCfg.Inputs {
			def, ok := allowedInputs[key]
			if !ok {
				continue // already reported as unknown
			}
			errors = append(errors, validateInputValue(key, val, def)...)
		}
	}

	return errors
}

func validateInputValue(name string, val interface{}, def *config.InputDef) []ValidationError {
	var errors []ValidationError
	field := fmt.Sprintf("inputs.%s", name)

	switch def.Type {
	case "string":
		strVal, ok := val.(string)
		if !ok {
			errors = append(errors, ValidationError{Field: field, Message: "expected string value"})
			return errors
		}
		if len(def.Enum) > 0 {
			found := false
			for _, e := range def.Enum {
				if e == strVal {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, ValidationError{
					Field:   field,
					Message: fmt.Sprintf("value %q is not one of: %s", strVal, strings.Join(def.Enum, ", ")),
				})
			}
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			errors = append(errors, ValidationError{Field: field, Message: "expected boolean value"})
		}
	case "number":
		switch val.(type) {
		case int, float64:
			// ok
		default:
			errors = append(errors, ValidationError{Field: field, Message: "expected number value"})
		}
	case "list":
		if _, ok := val.([]interface{}); !ok {
			errors = append(errors, ValidationError{Field: field, Message: "expected list value"})
		}
	}

	return errors
}
