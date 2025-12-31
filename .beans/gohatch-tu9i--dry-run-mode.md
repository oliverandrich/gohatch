---
# gohatch-tu9i
title: Dry-run mode
status: completed
type: feature
priority: normal
created_at: 2025-12-31T14:40:04Z
updated_at: 2025-12-31T14:56:22Z
---

Add a `--dry-run` flag that previews what would be created without making any changes.

## Context

Users may want to see what gohatch will do before actually creating files, especially when using unfamiliar templates.

## Checklist

- [x] Add `--dry-run` flag to CLI
- [x] Implement dry-run logic that shows:
  - Source URL that would be cloned
  - Target directory that would be created
  - Module path rewrite that would occur (old → new)
  - Extra extensions that would be processed
- [x] Skip actual file operations when dry-run is enabled
- [x] Add tests for dry-run mode
- [x] Update CLI help text
- [x] Update README with dry-run examples