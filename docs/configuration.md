# Configuration

gitrepoforge uses three configuration files at different levels.

## Root Config — `.gitrepoforge-config`

Located at the workspace root. Tells gitrepoforge where to find the central config and how to behave.

```yaml
config_repo: "config-repo"        # Required. Relative path to the central config repo.
default_branch: "main"            # Required. Default branch name for all repos.
excludes:                          # Optional. Glob patterns for repos to skip.
  - "archived-*"
  - "legacy/*"
branch_prefix: "gitrepoforge/"    # Optional. Prefix for created branches. Default: "gitrepoforge/"
create_pr: true                    # Optional. Create a pull request after push. Default: false.
```

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `config_repo` | string | yes | — | Path to the repo containing `inputs/` and `outputs/` directories. |
| `default_branch` | string | yes | — | The default branch name used when checking out repos. |
| `excludes` | list of strings | no | `[]` | Glob patterns matched against repo directory names. |
| `branch_prefix` | string | no | `"gitrepoforge/"` | Prefix prepended to branch names created by apply/bootstrap. |
| `create_pr` | bool | no | `false` | Whether to open a PR via `gh pr create` after pushing. |

## Central Config — `inputs/` and `outputs/` Directories

Located in the config repo (the repo referenced by `config_repo`). Instead of a single YAML file, the central config uses a directory-based layout: one file per input definition and one file per output rule.

```
config-repo/
├── inputs/
│   ├── language.yaml
│   ├── enable_ci.yaml
│   └── team.yaml
└── outputs/
    ├── .github/
    │   └── workflows/
    │       └── ci.yml.gitrepoforge
    ├── CODEOWNERS.gitrepoforge
    ├── .eslintrc.json.gitrepoforge
    ├── legacy.txt.gitrepoforge
    └── README.md.gitrepoforge
```

### Input Definitions — `inputs/`

Each input is defined in its own YAML file under `inputs/`. The filename (without `.yaml`) becomes the input name.

**`inputs/language.yaml`**
```yaml
type: string
required: true
enum: ["go", "python", "node"]
description: "Primary language of the repo."
```

**`inputs/enable_ci.yaml`**
```yaml
type: boolean
default: "true"
description: "Whether to generate CI config."
```

**`inputs/team.yaml`**
```yaml
type: string
required: false
description: "Owning team name."
```

#### Input Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | yes | One of `string`, `boolean`, `number`, `list`. |
| `required` | bool | no | If `true`, every repo must provide this input. |
| `enum` | list of strings | no | Allowed values (only for `string` type). |
| `default` | string | no | Default value when the input is omitted. |
| `description` | string | no | Human-readable description. |

The input name is derived from the filename (e.g., `inputs/language.yaml` → input name `language`).

### Output Rules — `outputs/`

Each output rule is defined in its own YAML file under `outputs/`. The file path mirrors the target path in the managed repo, with a `.gitrepoforge` suffix. For example, the rule for `.github/workflows/ci.yml` lives at `outputs/.github/workflows/ci.yml.gitrepoforge`.

**`outputs/.github/workflows/ci.yml.gitrepoforge`**
```yaml
mode: create
condition: "enable_ci"
template: |
  name: CI
  on: [push]
  jobs:
    build:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - name: Build
          run: make build
```

**`outputs/CODEOWNERS.gitrepoforge`**
```yaml
template: "* @{{getInput .Inputs \"team\"}}"
```

**`outputs/.eslintrc.json.gitrepoforge`**
```yaml
condition: "language == \"node\""
template: |
  {
    "extends": "eslint:recommended"
  }
```

**`outputs/legacy.txt.gitrepoforge`**
```yaml
mode: delete
```

**`outputs/README.md.gitrepoforge`**
```yaml
mode: partial
blocks:
  - begin_marker: "<!-- BEGIN MANAGED -->"
    end_marker: "<!-- END MANAGED -->"
    template: |
      This section is managed by gitrepoforge.
      Language: {{.Inputs.language}}
```

#### Output Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | no | `create` (default), `delete`, or `partial`. |
| `condition` | string | no | Condition that must be true for this rule to apply. See [templates.md](templates.md#conditions). |
| `template` | string | no | Inline Go template string for the file content. Used with `create` mode. |
| `blocks` | list | no | Block rules for `partial` mode only. |

The target file path is derived from the output file's path by removing the `outputs/` prefix and the `.gitrepoforge` suffix.

#### Modes

| Mode | Description |
|------|-------------|
| `create` | The entire file is managed by gitrepoforge. If missing it is created; if different it is updated. This is the default. |
| `delete` | The file should not exist. If present it is removed. |
| `partial` | Only managed blocks within the file are controlled; the rest is user-maintained. |

### Block Rules (`blocks`)

Used with `mode: partial` to manage sections within an existing file.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `begin_marker` | string | yes | Start marker for the managed block. |
| `end_marker` | string | yes | End marker for the managed block. |
| `template` | string | no | Inline Go template string for block content. |

If the markers are not found in the target file, the block (markers + rendered content) is appended to the end of the file.

## Per-Repo Config — `.gitrepoforge`

Located at the root of each managed repository. Declares the repo name and provides input values.

```yaml
name: my-repo
inputs:
  language: go
  enable_ci: true
  team: platform
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | yes | Must match the repository's directory name. |
| `inputs` | map | no | Key-value pairs matching the input definitions in the central config. |

### Validation Rules

- `name` must match the repo's folder name.
- All `required` inputs must be present.
- Unknown inputs (not defined in the central config) are rejected.
- Input values must match their declared `type`.
- String inputs with `enum` must use one of the allowed values.
