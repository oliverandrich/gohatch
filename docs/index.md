# gohatch

A project scaffolding tool for Go, inspired by [gonew](https://go.dev/blog/gonew) with additional features for template authors and users.

## Features

- Clone templates from GitHub, GitLab, or any Git host
- Use local directories as templates
- Automatic module path rewriting in `go.mod` and all `.go` files
- Template variable substitution (`__VarName__` placeholders)
- Path renaming with variable placeholders
- Interactive prompting for missing variables
- Post-generation hooks via `.gohatch.toml`
- Binary file detection and skipping
- Strict mode for unset variable enforcement
- Dry-run mode with detailed preview
- Git repository initialization with initial commit
- Support for specific tags, branches, or commits

## Installation

### Homebrew (macOS/Linux)

```bash
brew install oliverandrich/tap/gohatch
```

### Go Install

```bash
go install github.com/oliverandrich/gohatch/cmd/gohatch@latest
```

### Binary Downloads

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/oliverandrich/gohatch/releases) page.

### Build from Source

```bash
git clone https://github.com/oliverandrich/gohatch.git
cd gohatch
go build -o gohatch ./cmd/gohatch
```

## Quick Start

Create a new Go project from a template in one command:

```bash
gohatch user/go-template github.com/me/myapp
```

This clones the template, rewrites the module path, substitutes variables, and initializes a git repository — all in one step.

## Requirements

- Go 1.26 or later (for building from source or `go install`)
- Git (for cloning remote templates)
