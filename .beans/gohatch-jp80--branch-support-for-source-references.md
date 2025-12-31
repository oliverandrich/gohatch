---
# gohatch-jp80
title: Branch support for source references
status: completed
type: feature
priority: normal
created_at: 2025-12-31T14:37:38Z
updated_at: 2025-12-31T15:08:33Z
---

Support `@branch` syntax (e.g., `user/repo@main`, `user/repo@develop`) in addition to the existing tag and commit hash support.

## Context

Currently, gohatch supports:
- Tags: `user/repo@v1.0.0`
- Commit hashes: `user/repo@abc1234`

But branch references like `@main` or `@feature-branch` are not supported.

## Checklist

- [x] Add `RemoteLister` interface to query remote refs
- [x] Add `resolveRefType()` to detect if version is tag, branch, or commit
- [x] Update `GitSource.Fetch()` to query remote and use correct ref type
- [x] Add tests with mock lister for tag/branch/commit scenarios
- [x] Update CLI help text with branch examples
- [x] Update README with branch syntax documentation