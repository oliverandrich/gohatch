---
# gohatch-64ci
title: Git repository initialization after scaffolding
status: completed
type: feature
priority: normal
created_at: 2025-12-31T15:47:14Z
updated_at: 2025-12-31T15:51:12Z
---

Initialize a new git repository with initial commit after scaffolding. Default behavior, can be disabled with --no-git-init. Uses go-git instead of shelling out.

## Checklist

- [x] Add `--no-git-init` CLI flag
- [x] Remove template's .git directory after fetch
- [x] Initialize git repo using go-git's PlainInit
- [x] Create initial commit with message "Initial commit."
- [x] Update dry-run output
- [x] Add tests for dry-run behavior
- [x] Update README