---
# gohatch-veu2
title: Split rewrite package into concept-based files
status: completed
type: task
priority: normal
created_at: 2026-01-01T10:17:59Z
updated_at: 2026-01-01T10:20:24Z
---

## Ziel
`internal/rewrite/rewrite.go` (453 Zeilen) in kompaktere Dateien aufteilen.

## Checklist

- [ ] `patterns.go` erstellen (parseFilePatterns, matchesFilePattern)
- [ ] `variables.go` erstellen (Variables, replaceVariablesInFile)
- [ ] `paths.go` erstellen (RenamePaths, collectPathsToRename, updatePathWithRenames)
- [ ] `rewrite.go` → `module.go` umbenennen und bereinigen
- [ ] Tests ausführen