# Overview

gitrepoforge is a command-line tool that audits and applies standard file patterns across Git repositories in a workspace.

## Purpose

When managing many repositories, it is common to need consistent configuration files (CI pipelines, linters, templates, etc.). gitrepoforge lets you define a desired state once in a central config repo and then validate or apply that state across all discovered repositories.

## High-Level Architecture

```
workspace/
├── .gitrepoforge-config        # Root config — points to the config repo
├── config-repo/
│   ├── gitrepoforge.yaml       # Central config — inputs + file rules
│   └── templates/              # Template files referenced by file rules
├── repo-a/
│   └── .gitrepoforge           # Per-repo config — name + input values
├── repo-b/
│   └── .gitrepoforge
└── ...
```

### Components

| Component | Description |
|-----------|-------------|
| **Discovery** | Scans the workspace for Git repositories, applying exclude patterns from the root config. |
| **Schema** | Validates each repo's `.gitrepoforge` file against the input definitions in the central config. |
| **Engine** | Renders templates and computes findings (create, update, delete, block_replace) for each repo. |
| **GitOps** | Creates branches, commits changes, pushes, and optionally opens pull requests via `gh`. |
| **Output** | Formats results as human-readable text or JSON. |

### Workflow

1. **Load** the root config (`.gitrepoforge-config`) and central config (`gitrepoforge.yaml`).
2. **Discover** Git repos in the workspace, excluding patterns from the root config.
3. For each repo that has a `.gitrepoforge` file:
   - **Validate** inputs against the central schema.
   - **Compute findings** by rendering templates and comparing to the current file state.
4. **Report** findings (`validate`) or **apply** them via Git operations (`apply` / `bootstrap`).

### Commands

- **`validate`** — Dry-run audit. Reports drift without making changes.
- **`apply`** — Applies the desired state, creating a branch, committing, pushing, and optionally opening a PR.
- **`bootstrap`** — Like apply, but intended for first-time setup of a repo.

See [commands.md](commands.md) for full details.
