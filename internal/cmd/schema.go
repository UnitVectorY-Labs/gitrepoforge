package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/output"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/schema"
)

func runSchema(version string, args []string) {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output in JSON format instead of YAML")
	outputFlag := fs.String("output", "", "Write schema to a file instead of stdout")
	fs.Parse(args)

	workspaceDir, err := os.Getwd()
	if err != nil {
		output.Error(fmt.Sprintf("failed to get working directory: %v", err))
		os.Exit(1)
	}

	rootCfg, err := config.LoadRootConfig(workspaceDir)
	if err != nil {
		output.Error(fmt.Sprintf("root config error: %v", err))
		os.Exit(1)
	}

	configRepoPath := rootCfg.ResolveConfigRepoPath(workspaceDir)
	centralCfg, err := config.LoadCentralConfig(configRepoPath)
	if err != nil {
		output.Error(fmt.Sprintf("central config error: %v", err))
		os.Exit(1)
	}

	jsonSchema := schema.GenerateJSONSchema(centralCfg)

	var rendered string
	if *jsonFlag {
		rendered, err = schema.RenderSchemaJSON(jsonSchema)
	} else {
		rendered, err = schema.RenderSchemaYAML(jsonSchema)
	}
	if err != nil {
		output.Error(fmt.Sprintf("failed to render schema: %v", err))
		os.Exit(1)
	}

	if *outputFlag != "" {
		if err := writeSchemaFile(*outputFlag, rendered); err != nil {
			output.Error(fmt.Sprintf("failed to write schema: %v", err))
			os.Exit(1)
		}
	} else {
		fmt.Print(rendered)
	}
}

func writeSchemaFile(path, content string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}
