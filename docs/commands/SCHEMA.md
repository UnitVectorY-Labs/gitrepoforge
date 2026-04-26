---
layout: default
title: gitrepoforge schema
parent: Commands
nav_order: 4
permalink: /commands/schema
---

# schema

Generates a JSON Schema for the `.gitrepoforge` per-repo config file based on the config definitions in the config repo.

```
gitrepoforge schema [flags]
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--json` | no | Output in JSON format instead of the default YAML. |
| `--output <path>` | no | Write the schema to a file instead of stdout. |

## Behavior

1. Loads the root config (`.gitrepoforge-config`) and config repo.
2. Reads all config definitions from `config/`.
3. Generates a JSON Schema (draft-07) that describes the valid structure for `.gitrepoforge` repo config files.
4. Outputs the schema in YAML format by default, or JSON when `--json` is specified.
5. The output is deterministic — given the same config definitions, the same schema is always generated.

## Schema Details

The generated schema includes the following:

- **Top-level fields**: `name` (string, required), `default_branch` (string, required), `manifest` (string, optional, default `.managedfiles`), and `config` (object).
- The `config` property mirrors the config definitions: types are mapped to JSON Schema types (`string` → `string`, `boolean` → `boolean`, `number` → `number`, `list` → `array`, `object` → `object`).
- Enum constraints, patterns, default values, descriptions, and required fields are preserved in the schema.
- `additionalProperties: false` is set on all objects to enforce strict validation.
