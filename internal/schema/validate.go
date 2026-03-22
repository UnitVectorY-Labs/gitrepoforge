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

	config.ApplyConfigDefaults(repoCfg, centralCfg)

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

	errors = append(errors, validateConfigMap("config", repoCfg.Config, centralCfg.Definitions, true)...)

	return errors
}

func validateConfigMap(field string, values map[string]interface{}, definitions []config.ConfigDefinition, topLevel bool) []ValidationError {
	var errors []ValidationError

	allowedConfig := make(map[string]*config.ConfigDefinition, len(definitions))
	for i := range definitions {
		allowedConfig[definitions[i].Name] = &definitions[i]
	}

	for _, def := range definitions {
		if !def.Required {
			continue
		}
		if values == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", field, def.Name),
				Message: "required config value is missing",
			})
			continue
		}
		if _, ok := values[def.Name]; !ok {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", field, def.Name),
				Message: "required config value is missing",
			})
		}
	}

	if values == nil {
		return errors
	}

	for key := range values {
		if topLevel && config.IsReservedConfigName(key) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", field, key),
				Message: "reserved top-level field name cannot be used in config",
			})
			continue
		}
		if _, ok := allowedConfig[key]; !ok {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.%s", field, key),
				Message: "unknown config value",
			})
		}
	}

	for key, val := range values {
		def, ok := allowedConfig[key]
		if !ok {
			continue
		}
		errors = append(errors, validateConfigValue(fmt.Sprintf("%s.%s", field, key), val, def)...)
	}

	return errors
}

func validateConfigValue(field string, val interface{}, def *config.ConfigDefinition) []ValidationError {
	var errors []ValidationError

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
	case "object":
		objectVal, ok := asConfigMap(val)
		if !ok {
			errors = append(errors, ValidationError{Field: field, Message: "expected object value"})
			return errors
		}
		errors = append(errors, validateConfigMap(field, objectVal, def.Attributes, false)...)
	default:
		errors = append(errors, ValidationError{Field: field, Message: fmt.Sprintf("unsupported config type %q", def.Type)})
	}

	return errors
}

func asConfigMap(value interface{}) (map[string]interface{}, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed, true
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, nestedValue := range typed {
			keyName, ok := key.(string)
			if !ok {
				return nil, false
			}
			result[keyName] = nestedValue
		}
		return result, true
	default:
		return nil, false
	}
}
