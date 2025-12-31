---
# gohatch-xcpn
title: Verbose mode
status: completed
type: feature
priority: low
created_at: 2025-12-31T14:40:04Z
updated_at: 2025-12-31T16:05:15Z
---

Add a `--verbose` flag for detailed output during the scaffolding process.

Note: No short form `-v` because it's already used for `--var`.

## Context

Currently, gohatch provides minimal output. A verbose mode would help users understand what's happening, especially for debugging or when something goes wrong.

## Checklist

- [x] Add `--verbose` flag to CLI
- [x] Show detailed progress during template processing
- [x] Show each file being rewritten with import changes
- [x] Show files with variable replacements
- [x] Add tests verifying verbose output
- [x] Update CLI help text and README