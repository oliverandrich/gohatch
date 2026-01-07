---
# gohatch-7hjk
title: 'Git-Initialisierung: main statt master'
status: completed
type: bug
priority: normal
created_at: 2026-01-07T13:07:36Z
updated_at: 2026-01-07T13:08:31Z
---

gohatch initialisiert neue Repositories mit 'master' als Branch-Namen. Der Standard sollte 'main' sein. Lösung: git.PlainInit durch git.PlainInitWithOptions mit DefaultBranch: 'main' ersetzen.