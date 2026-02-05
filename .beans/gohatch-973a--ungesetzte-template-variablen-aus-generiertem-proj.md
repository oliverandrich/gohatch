---
# gohatch-973a
title: Remove unset template variables from generated project
status: completed
type: feature
priority: normal
created_at: 2026-02-05T06:43:05Z
updated_at: 2026-02-05T13:02:41Z
---

When a template contains `__VarName__` placeholders that are not set via `-v` during scaffolding, they currently remain unchanged in the generated project.

Desired behavior: Unset variables should be removed (i.e. replaced with an empty string) so that no `__VarName__` artifacts remain in the final project.

## Open Questions

- Should there be a warning when unset variables are found?
- Should there be a flag to control this behavior (e.g. `--strict` to abort on unset variables)?
- Does this affect only file contents or also path renaming (`RenamePaths`)?

## Affected Code

- `internal/rewrite/variables.go` – `Variables()` function
- `internal/rewrite/paths.go` – `RenamePaths()` function
- Possibly CLI flags in `cmd/gohatch/`