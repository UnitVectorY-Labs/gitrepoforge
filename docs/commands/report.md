---
layout: default
title: report
parent: Commands
nav_order: 3
permalink: /commands/report
---

# report

Generates a markdown report showing what changes [`apply`](apply.md) would make, without actually making them. Changes are grouped by output file path and deduplicated so identical diffs across repos appear only once.

```
gitrepoforge report [flags]
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--repo <name>` | no | Target a single repo by its directory name. |
| `--output <path>` | no | Write the markdown report to a file instead of stdout. |

## Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo.
2. Discovers Git repos in the workspace (or targets the single `--repo`).
3. For each repo with a valid `.gitrepoforge` config, computes the findings that `apply` would produce.
4. Aggregates findings by output file path across all repos.
5. Deduplicates diffs so that repos receiving the same change share a single diff block.
6. Outputs a markdown report to stdout or to the file specified by `--output`.

## Report Structure

The generated markdown report contains:

1. **Repository Summary** – a table listing each repository that has changes and the number of changes.
2. **File Summary** – a table listing each output file, its operation, and the number of affected repositories.
3. **File Sections** – one section per output file path, showing the affected repositories and the deduplicated diffs.

Diffs are rendered using the markdown diff code block syntax (` ```diff `). When `collapse_diffs` is `true` (the default), each diff is wrapped in a collapsible `<details>` block. See [Root Config](../ROOT-CONFIG.md) for how to configure this.

When multiple repos have the same change to a file, the diffs are deduplicated based on the added and removed lines. Context lines that differ between repos (due to surrounding content) are ignored for deduplication purposes. If different repos have different changes to the same file, each unique change is shown separately with its own list of affected repositories.
