---
# gohatch-xcpn
title: Verbose mode
status: draft
type: feature
priority: low
created_at: 2025-12-31T14:40:04Z
updated_at: 2025-12-31T14:40:04Z
---

Add a `--verbose` / `-v` flag for detailed output during the scaffolding process.

## Context

Currently, gohatch provides minimal output. A verbose mode would help users understand what's happening, especially for debugging or when something goes wrong.

## Checklist

- [ ] Add `--verbose` / `-v` flag to CLI
- [ ] Show detailed progress during git clone
- [ ] List files being copied (for local sources)
- [ ] Show each file being rewritten with import changes
- [ ] Show extra extension files being processed
- [ ] Add tests verifying verbose output
- [ ] Update CLI help text