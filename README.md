# gohatch

A project scaffolding tool for Go, inspired by [gonew](https://go.dev/blog/gonew) with additional features.

## Features

- Clone templates from GitHub, Codeberg, or any Git host
- Use local directories as templates
- Automatic module path rewriting in `go.mod` and all `.go` files
- Template variable substitution (`__VarName__` → `Value`)
- Initialize git repository with initial commit (optional)
- Support for specific tags (`@v1.0.0`), branches (`@main`), or commits (`@abc1234`)
- Optional string replacement in additional file types

## Requirements

- Go 1.24 or later (for building from source or `go install`)
- Git (for cloning remote templates)

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap oliverandrich/tap https://codeberg.org/oliverandrich/homebrew-tap.git
brew install gohatch
```

### Go Install

```bash
go install codeberg.org/oliverandrich/gohatch@latest
```

### Binary Downloads

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://codeberg.org/oliverandrich/gohatch/releases) page.

### Build from Source

```bash
git clone https://codeberg.org/oliverandrich/gohatch.git
cd gohatch
go build -o gohatch ./cmd/gohatch
```

To install into your `$GOPATH/bin`:

```bash
go install ./cmd/gohatch
```

## Usage

```bash
gohatch [options] <source> <module> [directory]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `source` | Template source (see formats below) |
| `module` | New Go module path |
| `directory` | Output directory (optional, defaults to last element of module) |

### Options

| Flag | Description |
|------|-------------|
| `-e, --extension` | Additional file extensions or filenames for module replacement |
| `-v, --var` | Set template variable (e.g., `--var Author="Name"`) |
| `-f, --force` | Proceed even if template has no go.mod |
| `--no-git-init` | Skip git repository initialization |
| `--dry-run` | Show what would be done without making any changes |
| `--verbose` | Show detailed progress output |

### Source Formats

| Format | Example |
|--------|---------|
| GitHub shorthand | `user/repo` |
| Full URL | `github.com/user/repo` |
| Other Git hosts | `codeberg.org/user/repo` |
| Specific tag | `user/repo@v1.0.0` |
| Specific branch | `user/repo@main` |
| Specific commit | `user/repo@abc1234` |
| Local directory | `./my-template` |

**Note:** gohatch automatically detects whether the version is a tag, branch, or commit hash by querying the remote repository.

## Examples

Create a new project from a GitHub template:

```bash
gohatch user/go-template github.com/me/myapp
```

Use a specific tag:

```bash
gohatch user/go-template@v1.0.0 github.com/me/myapp
```

Use a specific branch:

```bash
gohatch user/go-template@main github.com/me/myapp
```

Specify output directory:

```bash
gohatch user/go-template github.com/me/myapp ./projects/myapp
```

Also replace module path in config files:

```bash
gohatch -e toml -e yaml user/go-template github.com/me/myapp
```

Replace in specific files (by exact name):

```bash
gohatch -e yml -e justfile -e Makefile user/go-template github.com/me/myapp
```

Use a local template:

```bash
gohatch ./my-template github.com/me/myapp
```

Preview what would be done (dry-run):

```bash
gohatch --dry-run user/go-template github.com/me/myapp
```

Use a non-Go template (skip go.mod validation):

```bash
gohatch --force user/non-go-template github.com/me/myapp
```

## Template Variables

Templates can include placeholders that get replaced during scaffolding. Placeholders use dunder-style syntax: `__VarName__`.

### Default Variables

| Variable | Default Value |
|----------|---------------|
| `ProjectName` | Output directory name |

### Setting Variables

Use the `-v` or `--var` flag to set variables:

```bash
gohatch --var Author="Oliver Andrich" user/go-template github.com/me/myapp
```

Multiple variables:

```bash
gohatch -v ProjectName=MyApp -v Author="Oliver Andrich" user/go-template github.com/me/myapp
```

### Template Example

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

Variables are replaced in `.go` files and any additional extensions or filenames specified with `-e`.

Each `-e` pattern is treated as both a potential filename and extension:
- `yml` matches files named `yml` AND files with `.yml` extension
- `justfile` matches files named `justfile` AND files with `.justfile` extension

### Path Renaming

Variables can also be used in directory and file names:

```
Template structure:              After scaffolding:
cmd/__ProjectName__/main.go  →   cmd/myapp/main.go
__ProjectName___test.go      →   myapp_test.go
```

This allows you to create templates where the directory structure adapts to the project name.

## License

This project is licensed under the [European Union Public License 1.2](https://eupl.eu/) (EUPL-1.2).
