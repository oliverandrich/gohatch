# Templates

This guide covers how to create your own gohatch templates.

## Template Structure

A gohatch template is any directory (or Git repository) containing a Go project. At minimum, it should have a `go.mod` file. gohatch rewrites the module path and import statements automatically.

```
my-template/
├── go.mod
├── go.sum
├── cmd/
│   └── __ProjectName__/
│       └── main.go
├── internal/
│   └── app.go
└── .gohatch.toml          # optional configuration
```

## Configuration File

Templates can include a `.gohatch.toml` configuration file to specify default settings. This eliminates the need for users to pass `-e` flags manually.

### Format

```toml
# Schema version (defaults to 1)
version = 1

# File extensions/names for module path and variable replacement
extensions = ["toml", "yaml", "justfile", "Makefile"]

# Post-generation hooks
[[hooks]]
name = "tidy modules"
command = "go mod tidy"

[[hooks]]
name = "install dependencies"
command = "go install ./..."
```

### Fields

| Field        | Type       | Description                                        |
| ------------ | ---------- | -------------------------------------------------- |
| `version`    | `int`      | Schema version, currently `1`                      |
| `extensions` | `string[]` | File extensions/names for substitution             |
| `hooks`      | `Hook[]`   | Post-generation hook commands                      |

The `.gohatch.toml` file is automatically removed from the output after processing. Use `--keep-config` to retain it.

## Hooks

Hooks are shell commands that run after scaffolding is complete. They execute in the generated project directory.

### Definition

```toml
[[hooks]]
name = "tidy modules"
command = "go mod tidy"

[[hooks]]
name = "setup project"
command = "echo Setting up __ProjectName__..."
```

### Variable Substitution

Hook commands support the same `__VarName__` placeholders as file contents. Variables are substituted before execution.

### Execution

- Hooks run sequentially in the order they are defined
- Each hook runs with a **5-minute timeout**
- Hooks execute via `sh -c` in the generated project directory
- stdout and stderr are forwarded to the terminal

### User Confirmation

Before running hooks, gohatch displays the resolved commands and asks for confirmation:

```
Post-generation hooks:
  1. tidy modules: go mod tidy
  2. setup project: echo Setting up myapp...

Run these hooks? (y/n)
```

### Non-Interactive Behavior

Hooks are **skipped** when:

- The `--no-hooks` flag is set
- stdin is not a terminal (e.g., in CI pipelines)

In both cases, a warning message is printed.

## Non-Go Templates

gohatch validates that the template contains a `go.mod` file by default. For non-Go templates, use `--force` to skip this validation:

```bash
gohatch --force user/non-go-template github.com/me/myapp
```

When `--force` is used, module path rewriting is skipped (since there is no `go.mod`), but variable substitution and path renaming still work.

## Best Practices

- Include a `go.mod` with a descriptive template module path (e.g., `github.com/user/go-template`)
- Use `__ProjectName__` in directory names for project-specific paths
- Specify commonly needed extensions in `.gohatch.toml` so users don't need `-e` flags
- Keep hooks lightweight — use them for `go mod tidy` or similar quick setup tasks
- Document your template's variables in the README so users know what to set
- Test your template by running `gohatch --dry-run` against it
