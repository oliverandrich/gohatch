# Template Variables

## Variable Syntax

Templates use dunder-style placeholders: `__VarName__`. These placeholders are replaced during scaffolding in file contents and path names.

## Default Variables

gohatch automatically sets these variables based on the module path:

| Variable      | Default Value                                                        |
| ------------- | -------------------------------------------------------------------- |
| `ProjectName` | Output directory name                                                |
| `GitUser`     | Second path element of module (e.g., `github.com/user/repo` → `user`) |

Both can be overridden with `--var`.

## Setting Variables

Use the `-v` or `--var` flag to set variables:

```bash
gohatch --var Author="Oliver Andrich" user/go-template github.com/me/myapp
```

Multiple variables:

```bash
gohatch -v ProjectName=MyApp -v Author="Oliver Andrich" user/go-template github.com/me/myapp
```

### Example

In your template files:

```go
package main

const AppName = "__ProjectName__"
const Author = "__Author__"
```

After scaffolding with `--var Author="Oliver"`:

```go
package main

const AppName = "myapp"
const Author = "Oliver"
```

## Path Renaming

Variables can also be used in directory and file names:

```
Template structure:              After scaffolding:
cmd/__ProjectName__/main.go  →   cmd/myapp/main.go
__ProjectName___test.go      →   myapp_test.go
```

This allows templates where the directory structure adapts to the project name.

## Extension Matching

By default, variable substitution and module path rewriting apply to `.go` files and `go.mod`. Use `-e` to include additional file types.

Each `-e` pattern is treated as both a potential filename and extension:

- `-e yml` matches files named `yml` **and** files with `.yml` extension
- `-e justfile` matches files named `justfile` **and** files with `.justfile` extension

Templates can also specify default extensions in `.gohatch.toml` (see [Creating Templates](creating-templates.md)). CLI extensions are merged with config extensions.

## Interactive Prompting

When gohatch detects unset variables in your template, it prompts you interactively to provide values — if running in a terminal.

- Variables used in **paths** are marked as required (you must provide a value)
- Variables used only in **file contents** are optional (press Enter to skip)

Prompting is skipped when:

- The `--no-prompt` flag is set
- stdin is not a terminal (e.g., in CI pipelines)

## Unset Variable Behavior

gohatch scans the template for variable placeholders that have no value assigned:

- **In paths:** Always an error. You must set the variable with `--var` or provide it when prompted.
- **In file contents (default):** A warning is shown and the placeholder is removed (replaced with empty string).
- **In file contents (`--strict`):** Treated as an error. Scaffolding is aborted.

## Binary File Detection

gohatch automatically detects binary files and skips them during variable substitution and module path rewriting. This prevents corruption of images, compiled assets, and other non-text files in your template.
