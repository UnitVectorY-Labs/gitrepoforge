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
	if repoCfg.DefaultBranch == "" {
		errors = append(errors, ValidationError{Field: "default_branch", Message: "default_branch is required"})
	}

	allowedConfig := make(map[string]*config.ConfigDefinition)
	for i := range centralCfg.Definitions {
		allowedConfig[centralCfg.Definitions[i].Name] = &centralCfg.Definitions[i]
	}

	for _, def := range centralCfg.Definitions {
		if def.Required {
			if repoCfg.Config == nil {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("config.%s", def.Name),
					Message: "required config value is missing",
				})
			} else if _, ok := repoCfg.Config[def.Name]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("config.%s", def.Name),
					Message: "required config value is missing",
				})
			}
		}
	}

	if repoCfg.Config != nil {
		for key := range repoCfg.Config {
			if _, ok := allowedConfig[key]; !ok {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("config.%s", key),
					Message: "unknown config value",
				})
			}
		}
	}

	if repoCfg.Config != nil {
		for key, val := range repoCfg.Config {
			def, ok := allowedConfig[key]
			if !ok {
				continue
			}
			errors = append(errors, validateConfigValue(key, val, def)...)
		}
	}

	return errors
}

func validateConfigValue(name string, val interface{}, def *config.ConfigDefinition) []ValidationError {
	var errors []ValidationError
	field := fmt.Sprintf("config.%s", name)

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
	default:
		errors = append(errors, ValidationError{Field: field, Message: fmt.Sprintf("unsupported config type %q", def.Type)})
	}

	return errors
}
