package schema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
)

func TestGenerateJSONSchemaBasic(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Required: true, Enum: []string{"mit", "apache-2.0"}},
			{Name: "enabled", Type: "boolean"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	if schema.Schema != "http://json-schema.org/draft-07/schema#" {
		t.Fatalf("$schema = %q, want draft-07", schema.Schema)
	}
	if schema.Type != "object" {
		t.Fatalf("type = %q, want object", schema.Type)
	}
	if schema.AdditionalProperties {
		t.Fatal("additionalProperties should be false")
	}

	// config should be required because license is required
	wantRequired := []string{"config", "default_branch", "name"}
	if len(schema.Required) != len(wantRequired) {
		t.Fatalf("required = %v, want %v", schema.Required, wantRequired)
	}
	for i, r := range schema.Required {
		if r != wantRequired[i] {
			t.Fatalf("required[%d] = %q, want %q", i, r, wantRequired[i])
		}
	}

	// Check config.license property
	configProp := schema.Properties["config"]
	licenseProp, ok := configProp.Properties["license"]
	if !ok {
		t.Fatal("missing config.license property")
	}
	if licenseProp.Type != "string" {
		t.Fatalf("license type = %q, want string", licenseProp.Type)
	}
	if len(licenseProp.Enum) != 2 || licenseProp.Enum[0] != "mit" || licenseProp.Enum[1] != "apache-2.0" {
		t.Fatalf("license enum = %v, want [mit apache-2.0]", licenseProp.Enum)
	}

	// Check config.enabled property
	enabledProp, ok := configProp.Properties["enabled"]
	if !ok {
		t.Fatal("missing config.enabled property")
	}
	if enabledProp.Type != "boolean" {
		t.Fatalf("enabled type = %q, want boolean", enabledProp.Type)
	}

	// Check required in config
	if len(configProp.Required) != 1 || configProp.Required[0] != "license" {
		t.Fatalf("config required = %v, want [license]", configProp.Required)
	}
}

func TestGenerateJSONSchemaNoRequired(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "enabled", Type: "boolean"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	// config should NOT be required since no definitions are required
	wantRequired := []string{"default_branch", "name"}
	if len(schema.Required) != len(wantRequired) {
		t.Fatalf("required = %v, want %v", schema.Required, wantRequired)
	}
}

func TestGenerateJSONSchemaNestedObject(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{
				Name:     "docs",
				Type:     "object",
				Required: true,
				Attributes: []config.ConfigDefinition{
					{Name: "enabled", Type: "boolean", Default: true, HasDefault: true},
					{Name: "domain", Type: "string", Required: true},
				},
			},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	configProp := schema.Properties["config"]
	docsProp, ok := configProp.Properties["docs"]
	if !ok {
		t.Fatal("missing config.docs property")
	}
	if docsProp.Type != "object" {
		t.Fatalf("docs type = %q, want object", docsProp.Type)
	}

	enabledProp, ok := docsProp.Properties["enabled"]
	if !ok {
		t.Fatal("missing docs.enabled property")
	}
	if enabledProp.Type != "boolean" {
		t.Fatalf("docs.enabled type = %q, want boolean", enabledProp.Type)
	}
	if enabledProp.Default != true {
		t.Fatalf("docs.enabled default = %v, want true", enabledProp.Default)
	}

	domainProp, ok := docsProp.Properties["domain"]
	if !ok {
		t.Fatal("missing docs.domain property")
	}
	if domainProp.Type != "string" {
		t.Fatalf("docs.domain type = %q, want string", domainProp.Type)
	}

	if len(docsProp.Required) != 1 || docsProp.Required[0] != "domain" {
		t.Fatalf("docs required = %v, want [domain]", docsProp.Required)
	}
}

func TestGenerateJSONSchemaWithPattern(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "version", Type: "string", Pattern: `^(?P<major>\d+)\.(?P<minor>\d+)$`},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	configProp := schema.Properties["config"]
	versionProp, ok := configProp.Properties["version"]
	if !ok {
		t.Fatal("missing config.version property")
	}
	if versionProp.Pattern != `^(?P<major>\d+)\.(?P<minor>\d+)$` {
		t.Fatalf("version pattern = %q", versionProp.Pattern)
	}
}

func TestGenerateJSONSchemaWithDefault(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Default: "mit", HasDefault: true},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	configProp := schema.Properties["config"]
	licenseProp := configProp.Properties["license"]
	if licenseProp.Default != "mit" {
		t.Fatalf("license default = %v, want mit", licenseProp.Default)
	}
}

func TestGenerateJSONSchemaListType(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "tags", Type: "list"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	configProp := schema.Properties["config"]
	tagsProp, ok := configProp.Properties["tags"]
	if !ok {
		t.Fatal("missing config.tags property")
	}
	if tagsProp.Type != "array" {
		t.Fatalf("tags type = %q, want array", tagsProp.Type)
	}
}

func TestGenerateJSONSchemaNumberType(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "count", Type: "number"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	configProp := schema.Properties["config"]
	countProp, ok := configProp.Properties["count"]
	if !ok {
		t.Fatal("missing config.count property")
	}
	if countProp.Type != "number" {
		t.Fatalf("count type = %q, want number", countProp.Type)
	}
}

func TestRenderSchemaJSONDeterministic(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Required: true, Enum: []string{"mit", "apache-2.0"}},
			{Name: "enabled", Type: "boolean"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	// Render multiple times and verify determinism
	first, err := RenderSchemaJSON(schema)
	if err != nil {
		t.Fatalf("RenderSchemaJSON error: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := RenderSchemaJSON(schema)
		if err != nil {
			t.Fatalf("RenderSchemaJSON error on iteration %d: %v", i, err)
		}
		if again != first {
			t.Fatalf("JSON output not deterministic on iteration %d", i)
		}
	}

	// Verify valid JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(first), &parsed); err != nil {
		t.Fatalf("rendered JSON is not valid: %v", err)
	}

	// Verify it ends with newline
	if !strings.HasSuffix(first, "\n") {
		t.Fatal("JSON output should end with newline")
	}
}

func TestRenderSchemaYAMLDeterministic(t *testing.T) {
	centralCfg := &config.CentralConfig{
		Definitions: []config.ConfigDefinition{
			{Name: "license", Type: "string", Required: true, Enum: []string{"mit", "apache-2.0"}},
			{Name: "enabled", Type: "boolean"},
		},
	}

	schema := GenerateJSONSchema(centralCfg)

	// Render multiple times and verify determinism
	first, err := RenderSchemaYAML(schema)
	if err != nil {
		t.Fatalf("RenderSchemaYAML error: %v", err)
	}
	for i := 0; i < 10; i++ {
		again, err := RenderSchemaYAML(schema)
		if err != nil {
			t.Fatalf("RenderSchemaYAML error on iteration %d: %v", i, err)
		}
		if again != first {
			t.Fatalf("YAML output not deterministic on iteration %d", i)
		}
	}

	// Verify it ends with newline
	if !strings.HasSuffix(first, "\n") {
		t.Fatal("YAML output should end with newline")
	}
}
