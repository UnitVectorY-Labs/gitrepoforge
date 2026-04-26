---
layout: default
title: gitrepoforge
nav_order: 1
permalink: /
---

# gitrepoforge

gitrepoforge is a command-line tool that audits and applies standard file patterns across Git repositories in a workspace.

## Purpose

When managing many repositories, it is common to need consistent files such as licenses, CI definitions, or shared metadata. gitrepoforge lets you define that state once in a config repo and then validate or apply it across all discovered repositories.

## High-Level Architecture

```
workspace/
├── .gitrepoforge-config        # Root config points to the config repo and shared Git automation
├── config-repo/
│   ├── config/                 # Config definitions, one YAML file per key
│   │   └── license.yaml
│   ├── outputs/                # Output rules, path mirrors target with .gitrepoforge suffix
│   │   └── LICENSE.gitrepoforge
│   └── templates/              # Template files referenced by outputs
│       └── licenses/
│           ├── apache-2.0.tmpl
│           └── mit.tmpl
├── repo-a/
│   ├── .gitrepoforge           # Per-repo config with name, default branch, and config values
│   └── .managedfiles.yaml
├── repo-b/
│   ├── .gitrepoforge
│   └── .managedfiles.yaml
└── ...
```

### Components

| Component | Description |
|-----------|-------------|
| **Discovery** | Scans the workspace for Git repositories, applying exclude patterns from the root config. |
| **Schema** | Validates each repo's `.gitrepoforge` file against the config definitions in the config repo. |
| **Engine** | Selects a template file for each output rule, renders it, and computes findings (`create`, `update`, `delete`). |
| **GitOps** | Creates branches, commits changes, pushes, and optionally opens pull requests via `gh`. |
| **Output** | Formats results as human-readable text or JSON. |

### Workflow

1. **Load** the root config (`.gitrepoforge-config`) and config repo (`config/`, `outputs/`, `templates/`).
2. **Discover** Git repos in the workspace, excluding patterns from the root config.
3. For each repo that has a `.gitrepoforge` file:
   - **Validate** repo metadata and config values against the shared schema.
   - **Compute findings** by selecting a matching template and comparing the rendered file to disk.
4. **Maintain** `.managedfiles.yaml` so each repo has a generated inventory of files and managed sections.
5. **Report** findings (`validate`) or **apply** them, optionally followed by root-configured Git operations (`apply`).

### Commands

- **`validate`**: Dry-run audit. Reports drift without making changes.
- **`apply`**: Applies the desired state and, when configured, runs shared Git automation.
- **`report`**: Generates a markdown report of what `apply` would change, without making changes.
- **`schema`**: Generates a JSON Schema for the `.gitrepoforge` per-repo config file.

See [Commands](/commands) for full details.
