---
# gohatch-a7hy
title: Template validation
status: draft
type: feature
priority: low
created_at: 2025-12-31T14:40:04Z
updated_at: 2025-12-31T14:40:04Z
---

Validate that the source is a valid Go project (contains go.mod) before proceeding with the full clone.

## Context

If a user accidentally points to a non-Go repository, gohatch will clone it and then silently skip the module rewrite. Early validation would provide a better user experience.

## Considerations

- For git sources: Could do a shallow clone, check for go.mod, then abort if missing
- For local sources: Easy to check before copying
- Should this be a warning or an error?
- Maybe add a `--force` flag to proceed anyway for non-Go templates

## Checklist

- [ ] Add validation check after fetching source
- [ ] Provide clear error message if go.mod is missing
- [ ] Consider adding `--force` flag to skip validation
- [ ] Add tests for validation scenarios
- [ ] Update CLI help text if --force is added