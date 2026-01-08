---
# gohatch-0kqm
title: Template-Konfigurationsdatei (.gohatch.toml)
status: completed
type: feature
priority: normal
created_at: 2026-01-08T08:22:28Z
updated_at: 2026-01-08T08:28:28Z
---

Einführung von .gohatch.toml als Template-Konfigurationsdatei für Extensions.

## Checklist
- [x] internal/config/config.go erstellen (Structs)
- [x] internal/config/load.go erstellen (TOML Parsing)
- [x] internal/config/remove.go erstellen
- [x] internal/config/config_test.go erstellen
- [x] go.mod aktualisieren (BurntSushi/toml)
- [x] cmd/gohatch/main.go anpassen (keepConfig Flag, mergeExtensions, executeScaffold)
- [x] Dry-Run Output anpassen
- [x] Tests erweitern (Integration-Tests in main_test.go)