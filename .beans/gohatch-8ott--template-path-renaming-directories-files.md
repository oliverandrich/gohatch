---
# gohatch-8ott
title: Template path renaming (directories & files)
status: completed
type: feature
priority: normal
created_at: 2025-12-31T17:04:07Z
updated_at: 2025-12-31T17:05:12Z
---

Support template variables (`__VarName__`) in directory and file names, not just file contents.

## Example

```
Template:                      After scaffolding:
cmd/__ProjectName__/main.go    cmd/myapp/main.go
```

## Implementation

Two-phase approach:
1. **Collect**: Walk tree, collect all paths containing `__VarName__`
2. **Rename**: Sort by depth (deepest first), rename from inside out

## Checklist

- [x] `RenamePaths()` function in `internal/rewrite/rewrite.go`
- [x] `collectPathsToRename()` helper
- [x] Sort by depth (deepest first)
- [x] Integration in `cmd/gohatch/main.go` (before module rewrite)
- [x] Tests for all scenarios
- [x] Verbose output for renames
- [x] Update README

## Test Scenarios

1. Simple directory: `cmd/__ProjectName__/` → `cmd/myapp/`
2. Nested: `__A__/__B__/file.go` → `x/y/file.go`
3. File rename: `__ProjectName__.go` → `myapp.go`
4. Combination: directory + file + content
5. No matches: no changes