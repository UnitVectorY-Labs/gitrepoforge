package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/gitrepoforge/internal/config"
	"github.com/UnitVectorY-Labs/gitrepoforge/internal/schema"
)

func TestScenarioTestdata(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "testdata"))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		caseDir := filepath.Join(root, entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			workspace := t.TempDir()
			copyDir(t, filepath.Join(caseDir, "scenario"), workspace)

			configRepoPath := filepath.Join(workspace, "config-repo")
			repoPath := filepath.Join(workspace, "repo")

			centralCfg, err := config.LoadCentralConfig(configRepoPath)
			if expectedError := readOptionalFile(t, filepath.Join(caseDir, "error.txt")); expectedError != "" {
				stage := strings.TrimSpace(readOptionalFile(t, filepath.Join(caseDir, "error-stage.txt")))
				if stage == "" {
					stage = "load-central"
				}
				assertExpectedError(t, stage, expectedError, err, repoPath, centralCfg)
				return
			}
			if err != nil {
				t.Fatalf("LoadCentralConfig returned error: %v", err)
			}

			repoCfg, err := config.LoadRepoConfig(repoPath)
			if err != nil {
				t.Fatalf("LoadRepoConfig returned error: %v", err)
			}

			if expectedValidation := readOptionalFile(t, filepath.Join(caseDir, "validation-errors.txt")); expectedValidation != "" {
				var got []string
				for _, validationErr := range schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath) {
					got = append(got, validationErr.Error())
				}

				want := nonEmptyLines(expectedValidation)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("validation errors = %v, want %v", got, want)
				}
				return
			}

			validationErrs := schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath)
			if len(validationErrs) != 0 {
				t.Fatalf("unexpected validation errors: %v", validationErrs)
			}

			findings, err := ComputeFindings(repoCfg, centralCfg, repoPath)
			if err != nil {
				t.Fatalf("ComputeFindings returned error: %v", err)
			}
			if err := ApplyFindings(findings, repoPath); err != nil {
				t.Fatalf("ApplyFindings returned error: %v", err)
			}

			assertDirectoryMatches(t, filepath.Join(caseDir, "expected", "repo"), repoPath)

			// Assert schema generation for valid test cases
			assertSchemaMatches(t, caseDir, centralCfg)
		})
	}
}

func assertExpectedError(t *testing.T, stage, expected string, loadErr error, repoPath string, centralCfg *config.CentralConfig) {
	t.Helper()

	switch stage {
	case "load-central":
		if loadErr == nil {
			t.Fatal("expected central config load error, got nil")
		}
		if !strings.Contains(loadErr.Error(), strings.TrimSpace(expected)) {
			t.Fatalf("error %q does not contain %q", loadErr.Error(), strings.TrimSpace(expected))
		}
	case "compute":
		if loadErr != nil {
			t.Fatalf("LoadCentralConfig returned error: %v", loadErr)
		}
		repoCfg, err := config.LoadRepoConfig(repoPath)
		if err != nil {
			t.Fatalf("LoadRepoConfig returned error: %v", err)
		}
		if errs := schema.ValidateRepoConfig(repoCfg, centralCfg, repoPath); len(errs) != 0 {
			t.Fatalf("unexpected validation errors: %v", errs)
		}
		_, err = ComputeFindings(repoCfg, centralCfg, repoPath)
		if err == nil {
			t.Fatal("expected compute error, got nil")
		}
		if !strings.Contains(err.Error(), strings.TrimSpace(expected)) {
			t.Fatalf("error %q does not contain %q", err.Error(), strings.TrimSpace(expected))
		}
	default:
		t.Fatalf("unsupported error stage %q", stage)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read %s: %v", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				t.Fatalf("failed to create %s: %v", dstPath, err)
			}
			copyDir(t, srcPath, dstPath)
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", srcPath, err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			t.Fatalf("failed to create parent dirs for %s: %v", dstPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", dstPath, err)
		}
	}
}

func assertDirectoryMatches(t *testing.T, expectedDir, actualDir string) {
	t.Helper()

	expectedFiles := listFiles(t, expectedDir)
	actualFiles := listFiles(t, actualDir)

	if !reflect.DeepEqual(expectedFiles, actualFiles) {
		t.Fatalf("file list mismatch: got %v want %v", actualFiles, expectedFiles)
	}

	for _, relPath := range expectedFiles {
		expectedData, err := os.ReadFile(filepath.Join(expectedDir, relPath))
		if err != nil {
			t.Fatalf("failed to read expected file %s: %v", relPath, err)
		}
		actualData, err := os.ReadFile(filepath.Join(actualDir, relPath))
		if err != nil {
			t.Fatalf("failed to read actual file %s: %v", relPath, err)
		}
		if string(actualData) != string(expectedData) {
			t.Fatalf("content mismatch for %s:\nactual: %q\nwant:   %q", relPath, string(actualData), string(expectedData))
		}
	}
}

func listFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}

	slices.Sort(files)
	return files
}

func readOptionalFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}

func nonEmptyLines(content string) []string {
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func assertSchemaMatches(t *testing.T, caseDir string, centralCfg *config.CentralConfig) {
	t.Helper()

	jsonSchema := schema.GenerateJSONSchema(centralCfg)

	// Assert YAML schema
	expectedYAML := readOptionalFile(t, filepath.Join(caseDir, "expected-schema.yaml"))
	if expectedYAML != "" {
		gotYAML, err := schema.RenderSchemaYAML(jsonSchema)
		if err != nil {
			t.Fatalf("RenderSchemaYAML error: %v", err)
		}
		if gotYAML != expectedYAML {
			t.Fatalf("YAML schema mismatch:\ngot:\n%s\nwant:\n%s", gotYAML, expectedYAML)
		}

		// Verify determinism
		gotYAML2, err := schema.RenderSchemaYAML(jsonSchema)
		if err != nil {
			t.Fatalf("RenderSchemaYAML error on second call: %v", err)
		}
		if gotYAML2 != gotYAML {
			t.Fatal("YAML schema output is not deterministic")
		}
	}

	// Assert JSON schema
	expectedJSON := readOptionalFile(t, filepath.Join(caseDir, "expected-schema.json"))
	if expectedJSON != "" {
		gotJSON, err := schema.RenderSchemaJSON(jsonSchema)
		if err != nil {
			t.Fatalf("RenderSchemaJSON error: %v", err)
		}
		if gotJSON != expectedJSON {
			t.Fatalf("JSON schema mismatch:\ngot:\n%s\nwant:\n%s", gotJSON, expectedJSON)
		}

		// Verify determinism
		gotJSON2, err := schema.RenderSchemaJSON(jsonSchema)
		if err != nil {
			t.Fatalf("RenderSchemaJSON error on second call: %v", err)
		}
		if gotJSON2 != gotJSON {
			t.Fatal("JSON schema output is not deterministic")
		}
	}
}
