---
# gohatch-3tkn
title: Replace GitSource mocks with real bare repo tests
status: completed
type: task
priority: normal
created_at: 2026-01-01T09:07:59Z
updated_at: 2026-01-01T09:07:59Z
---

## Ziel
Mock-basierte Tests durch echte Git-Operationen mit lokalen Bare-Repos ersetzen.

## Checklist

- [x] Test-Helper `setupBareRepo` erstellen (+ Varianten für Tag, Branch, Commits)
- [x] Interfaces in source.go vereinfachen (Repository, Worktree, gitRepository, GitCloner, RemoteLister entfernt)
- [x] Mock-Tests durch echte Bare-Repo-Tests ersetzen
- [x] Mock-Typen entfernen (mockCloner, mockRepository, mockWorktree, mockLister)
- [x] Tests ausführen und verifizieren
