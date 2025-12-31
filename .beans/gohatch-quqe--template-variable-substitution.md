---
# gohatch-quqe
title: Template variable substitution
status: draft
type: feature
priority: normal
created_at: 2025-12-31T14:46:51Z
updated_at: 2025-12-31T14:46:51Z
---

Replace template variables in files during scaffolding, similar to cookiecutter but simpler.

## Scope

Simple variable substitution without interactive prompts or complex logic. Initial variables:
- `__ProjectName__` – basename of the target directory (default), overridable via CLI
- `__Author__` – must be provided via CLI (no default)

## Syntax

Use dunder-style placeholders: `__VariableName__`
- Simple string replacement, no template engine
- No conflicts with Go templates or other code
- Easy to spot in files

## CLI Interface

```bash
# Use defaults (ProjectName = myapp from path)
gohatch user/template github.com/me/myapp

# Override ProjectName
gohatch --var ProjectName=MyApp user/template github.com/me/myapp

# Multiple variables
gohatch --var ProjectName=MyApp --var Author="Oliver Andrich" user/template github.com/me/myapp

# Short form
gohatch -V ProjectName=MyApp -V Author="Oliver Andrich" user/template github.com/me/myapp
```

## Files to Process

- All `.go` files (default, same as module rewrite)
- Additional extensions via `-e` flag (same as module rewrite)
- Reuse existing extension selection logic

## Checklist

- [ ] Add `--var` / `-V` flag for key=value pairs (repeatable)
- [ ] Parse variables into map, set ProjectName default from directory basename
- [ ] Implement `replaceVariables(content, vars)` function
- [ ] Integrate variable replacement into `rewriteGoFile()` 
- [ ] Integrate variable replacement into `rewriteTextFile()`
- [ ] Add tests for variable parsing
- [ ] Add tests for variable replacement
- [ ] Add tests for default ProjectName behavior
- [ ] Update CLI help text with examples
- [ ] Update README with variable substitution documentation

## Future Considerations (out of scope for now)

- Template manifest file (gohatch.yaml) defining available variables
- Variable validation (required vs optional)
- Go template syntax as alternative/upgrade path
- File/directory renaming based on variables