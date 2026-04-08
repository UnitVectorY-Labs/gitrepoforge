package schema

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"gopkg.in/yaml.v3"
)

// JSONSchema represents a JSON Schema document.
type JSONSchema struct {
	Schema               string                `json:"$schema"`
	Type                 string                `json:"type"`
	AdditionalProperties bool                  `json:"additionalProperties"`
	Required             []string              `json:"required,omitempty"`
	Properties           map[string]SchemaNode `json:"properties"`
}

// SchemaNode represents a property in the JSON Schema.
type SchemaNode struct {
	Type                 string                `json:"type"`
	Description          string                `json:"description,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	Pattern              string                `json:"pattern,omitempty"`
	Default              interface{}           `json:"default,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Properties           map[string]SchemaNode `json:"properties,omitempty"`
	Items                *SchemaNode           `json:"items,omitempty"`
}

// GenerateJSONSchema produces a deterministic JSON Schema for the .gitrepoforge
// repo config file based on the central config definitions.
func GenerateJSONSchema(centralCfg *config.CentralConfig) *JSONSchema {
	schema := &JSONSchema{
		Schema:               "http://json-schema.org/draft-07/schema#",
		Type:                 "object",
		AdditionalProperties: false,
		Required:             []string{"default_branch", "name"},
		Properties: map[string]SchemaNode{
			"name": {
				Type:        "string",
				Description: "Must match the repository folder name.",
			},
			"default_branch": {
				Type:        "string",
				Description: "The default branch of the repository.",
			},
		},
	}

	configNode := buildConfigNode(centralCfg.Definitions)
	schema.Properties["config"] = configNode

	if hasRequiredDefinitions(centralCfg.Definitions) {
		schema.Required = []string{"config", "default_branch", "name"}
	}

	return schema
}

func buildConfigNode(definitions []config.ConfigDefinition) SchemaNode {
	node := SchemaNode{
		Type:       "object",
		Properties: make(map[string]SchemaNode),
	}
	additionalProperties := false
	node.AdditionalProperties = &additionalProperties

	var required []string
	for _, def := range definitions {
		propNode := definitionToSchemaNode(def)
		node.Properties[def.Name] = propNode
		if def.Required {
			required = append(required, def.Name)
		}
	}

	sort.Strings(required)
	if len(required) > 0 {
		node.Required = required
	}

	return node
}

func definitionToSchemaNode(def config.ConfigDefinition) SchemaNode {
	node := SchemaNode{
		Description: def.Description,
	}

	switch def.Type {
	case "string":
		node.Type = "string"
		if len(def.Enum) > 0 {
			node.Enum = def.Enum
		}
		if def.Pattern != "" {
			node.Pattern = def.Pattern
		}
	case "boolean":
		node.Type = "boolean"
	case "number":
		node.Type = "number"
	case "list":
		node.Type = "array"
	case "object":
		node.Type = "object"
		if len(def.Attributes) > 0 {
			node.Properties = make(map[string]SchemaNode)
			additionalProperties := false
			node.AdditionalProperties = &additionalProperties

			var required []string
			for _, attr := range def.Attributes {
				attrNode := definitionToSchemaNode(attr)
				node.Properties[attr.Name] = attrNode
				if attr.Required {
					required = append(required, attr.Name)
				}
			}
			sort.Strings(required)
			if len(required) > 0 {
				node.Required = required
			}
		}
	}

	if def.HasDefault {
		node.Default = def.Default
	}

	return node
}

func hasRequiredDefinitions(definitions []config.ConfigDefinition) bool {
	for _, def := range definitions {
		if def.Required {
			return true
		}
	}
	return false
}

// RenderSchemaJSON renders the JSON Schema as deterministic JSON with indentation.
func RenderSchemaJSON(schema *JSONSchema) (string, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}
	return string(data) + "\n", nil
}

// RenderSchemaYAML renders the JSON Schema as deterministic YAML.
func RenderSchemaYAML(schema *JSONSchema) (string, error) {
	// Round-trip through JSON to get a deterministic ordered map structure
	jsonData, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	// Build a yaml.Node tree from JSON for deterministic key ordering
	node, err := jsonToYAMLNode(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to convert schema to YAML node: %w", err)
	}

	data, err := yaml.Marshal(node)
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema to YAML: %w", err)
	}
	return string(data), nil
}

// jsonToYAMLNode converts JSON bytes into a yaml.Node tree with sorted map keys
// for deterministic output.
func jsonToYAMLNode(data []byte) (*yaml.Node, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return valueToYAMLNode(raw), nil
}

func valueToYAMLNode(v interface{}) *yaml.Node {
	switch val := v.(type) {
	case map[string]interface{}:
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: k,
			}
			valNode := valueToYAMLNode(val[k])
			node.Content = append(node.Content, keyNode, valNode)
		}
		return node
	case []interface{}:
		node := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
		}
		for _, item := range val {
			node.Content = append(node.Content, valueToYAMLNode(item))
		}
		return node
	case string:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: val,
		}
	case float64:
		// JSON numbers are float64; format as integer if possible
		if val == float64(int64(val)) {
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!int",
				Value: fmt.Sprintf("%d", int64(val)),
			}
		}
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!float",
			Value: fmt.Sprintf("%g", val),
		}
	case bool:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!bool",
			Value: fmt.Sprintf("%t", val),
		}
	case nil:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: "null",
		}
	default:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: fmt.Sprintf("%v", val),
		}
	}
}
