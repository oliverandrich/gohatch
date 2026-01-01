---
# gohatch-39iv
title: Migration von GitHub zu Codeberg
status: in-progress
type: task
created_at: 2026-01-02T18:17:54Z
updated_at: 2026-01-02T18:17:54Z
---

Migration des gohatch-Projekts von github.com zu codeberg.org.

## Checklist

### Im Projekt (Teil A)
- [x] Go-Modul umbenennen (go.mod, imports in main.go und main_test.go)
- [x] go mod tidy ausfuehren
- [x] README.md aktualisieren (Homebrew, Releases, Clone-URL, go install)
- [x] CLAUDE.md aktualisieren
- [x] .goreleaser.yml fuer Codeberg konfigurieren
- [x] .github/ Verzeichnis loeschen
- [x] .woodpecker.yml erstellen (CI)
- [x] .woodpecker/release.yml erstellen (Release-Pipeline)
- [x] Tests laufen lassen (go test -race ./...)

### Manuell (Teil B) - Hinweise fuer User
- [ ] Codeberg Repos erstellen (gohatch + homebrew-tap)
- [ ] SSH-Key und API-Token generieren
- [ ] Remote aendern und pushen
- [ ] Woodpecker aktivieren
- [ ] GitHub Repos loeschen