---
# gohatch-jp80
title: Branch support for source references
status: draft
type: feature
priority: normal
created_at: 2025-12-31T14:37:38Z
updated_at: 2025-12-31T14:37:38Z
---

Support `@branch` syntax (e.g., `user/repo@main`, `user/repo@develop`) in addition to the existing tag and commit hash support.

## Context

Currently, gohatch supports:
- Tags: `user/repo@v1.0.0`
- Commit hashes: `user/repo@abc1234`

But branch references like `@main` or `@feature-branch` are not supported.

## Checklist

- [ ] Update `isCommitHash()` to distinguish between commit hashes and branch names
- [ ] Add `isBranchName()` helper function or similar detection logic
- [ ] Update `GitSource.Fetch()` to handle branch references with shallow clone
- [ ] Add tests for branch reference parsing
- [ ] Add tests for branch fetching (with mocks)
- [ ] Update CLI help text with branch examples
- [ ] Update README with branch syntax documentation