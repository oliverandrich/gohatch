---
# gohatch-qp5t
title: Create go-cli-template project
status: completed
type: task
priority: normal
created_at: 2025-12-31T17:23:22Z
updated_at: 2025-12-31T17:51:15Z
---

Create a CLI template project at /Users/oa/Projekte/privat/go-cli-template based on gohatch.

Uses template variables:
- `__ProjectName__` for directory and binary names
- `__ProjectDescription__` for project description
- Module path gets replaced by gohatch

## Checklist

- [x] Create directory structure
- [x] go.mod with urfave/cli v3
- [x] cmd/__ProjectName__/main.go
- [x] internal/example/ package
- [x] justfile with standard tasks
- [x] .golangci.yml
- [x] .goreleaser.yml with template variables
- [x] .github/workflows (ci.yml, release.yml)
- [x] README.md
- [x] CLAUDE.md
- [x] LICENSE (EUPL-1.2)
- [x] .gitignore
- [x] .pre-commit-config.yaml
- [x] Git init and initial commit

## Additional Feature

Added support for filename patterns in `-e` flag (e.g., `-e justfile`, `-e Makefile`) in addition to extensions.