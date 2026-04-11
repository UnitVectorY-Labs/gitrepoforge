---
layout: default
title: Configuration
nav_order: 3
permalink: /configuration
has_children: true
---

# Configuration

gitrepoforge uses three configuration files at different levels to control behavior across a workspace.

## Configuration Files Overview

| Configuration | File | Location | Purpose |
|--------------|------|----------|---------|
| [Workspace Config](/configuration/workspace) | `.gitrepoforge-config` | Workspace root | Points to the config repo, defines exclude patterns, and configures Git automation. |
| [Per-Repo Config](/configuration/per-repo) | `.gitrepoforge` | Each managed repo root | Declares repo metadata and config values that feed into templates. |
| [Config Repository](/configuration/config-repo) | `config/`, `outputs/`, `templates/` | Config repo directory | Contains shared config definitions, output rules, and template files. |

## How They Work Together

```
workspace/
├── .gitrepoforge-config          ← Workspace Config
├── config-repo/                  ← Config Repository
│   ├── config/                      Shared config definitions
│   ├── outputs/                     Output rules
│   └── templates/                   Template files
├── repo-a/
│   └── .gitrepoforge             ← Per-Repo Config
├── repo-b/
│   └── .gitrepoforge             ← Per-Repo Config
└── ...
```

1. The **Workspace Config** (`.gitrepoforge-config`) tells gitrepoforge where to find the config repo and which repos to exclude.
2. The **Config Repository** defines the allowed config keys, output file rules, and templates.
3. Each **Per-Repo Config** (`.gitrepoforge`) provides repo-specific values that are validated against the config definitions and used to render templates.
