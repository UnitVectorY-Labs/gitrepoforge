# AGENTS.md

`gitrepoforge` is a Go command-line application that standardizes, audits, and applies managed files across Git repositories in a workspace.

## Repository Structure

- `main.go` — CLI entrypoint.
- `internal/cmd/` — Subcommands (`validate`, `apply`, `report`, `schema`), flag parsing, and top-level help.
- `internal/config/` — Loading and validation of workspace (`.gitrepoforge-config`), central (`config/`, `outputs/`, `templates/`), and per-repo (`.gitrepoforge`) configs.
- `internal/discovery/` — Workspace Git repository discovery and exclude matching.
- `internal/engine/` — Condition evaluation, template rendering, section directives, file filtering, managed-files manifest generation, and finding computation/application.
- `internal/gitops/` — Local `git` and GitHub CLI (`gh`) automation helpers.
- `internal/schema/` — Per-repo config validation and JSON Schema generation.
- `internal/diff/` and `internal/output/` — Unified diffs, human/JSON output, and Markdown reports.
- `testdata/` — Scenario fixtures verifying end-to-end engine and schema behavior.
- `docs/` — User-facing documentation site (Jekyll).

## Maintenance Guidelines

1. **Test and build every change**:
   - Add or update unit tests alongside changed packages (and `testdata/` fixtures when adding template, condition, or schema behavior).
   - Run `just test` (or `go test ./...`) and `just build` (or `go build ./...`) before opening a PR.
2. **Keep documentation in sync**:
   - Whenever adding or changing commands, flags, config fields, template behavior, or Git operations, update the relevant files in `docs/` (for example `docs/COMMANDS.md`, `docs/commands/*.md`, `docs/ROOT-CONFIG.md`, `docs/GITOPS.md`, `docs/MANAGED-FILES-MANIFEST.md`).
   - Update CLI help output in `internal/cmd/root.go` when flags or commands change.
3. **Respect managed files**:
   - Check `.managedfiles` before editing repository-level workflows, dotfiles, or managed sections of files such as `README.md` and `go.mod`, as those are managed by `gitrepoforge`.
4. **Keep dependencies minimal**:
   - Prefer the Go standard library (the module intentionally depends only on `gopkg.in/yaml.v3`).
