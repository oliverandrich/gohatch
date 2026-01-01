---
# gohatch-lipv
title: Refactor source.go structure and inline single-use helpers
status: completed
type: task
priority: normal
created_at: 2026-01-01T08:38:24Z
updated_at: 2026-01-01T08:40:04Z
---

## Problem

Die Datei `internal/source/source.go` hat mehrere strukturelle Probleme:

1. **Toter Code**: `isCommitHash()` (Zeile 144-157) wird nirgends verwendet
2. **Unlogische Reihenfolge**: Das zentrale `Source` Interface steht mitten zwischen Git-Abstraktionen
3. **Unnötige Indirektionen**: Mehrere private Funktionen werden nur einmal verwendet
4. **Fehlende Gruppierung**: Interfaces, Structs und Funktionen sind durcheinander gemischt

## Checklist

- [x] `isCommitHash()` entfernen (ungenutzt)
- [x] `copyDir()` in `LocalSource.Fetch()` inlinen
- [x] `removeGitDir()` in `GitSource.Fetch()` inlinen (ersetzt durch direkten `os.RemoveAll`-Aufruf)
- [x] Datei logisch neu strukturieren:
  - Zuerst das zentrale `Source` Interface
  - Dann `LocalSource` mit seiner Fetch-Methode
  - Dann `GitSource` mit allen Git-bezogenen Interfaces und Implementierungen
  - `Parse()` mit `splitVersion()` und `buildGitURL()` am Ende
- [x] Tests ausführen um sicherzustellen, dass nichts kaputt ist
