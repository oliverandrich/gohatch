# gohatch

A project scaffolding tool for Go, inspired by [gonew](https://go.dev/blog/gonew) with additional features.

## Features

- Clone templates from GitHub, Codeberg, or any Git host
- Use local directories as templates
- Automatic module path rewriting in `go.mod` and all `.go` files
- Support for specific tags (`@v1.0.0`) or commits (`@abc1234`)
- Optional string replacement in additional file types

## Requirements

- Go 1.25 or later (for building from source or `go install`)
- Git (for cloning remote templates)

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap oliverandrich/tap
brew install gohatch
```

Or as a single command:

```bash
brew install oliverandrich/tap/gohatch
```

### Go Install

```bash
go install github.com/oliverandrich/gohatch@latest
```

### Binary Downloads

Pre-built binaries for Linux, macOS, and Windows are available on the [Releases](https://github.com/oliverandrich/gohatch/releases) page.

### Build from Source

```bash
git clone https://github.com/oliverandrich/gohatch.git
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
| `-e, --extension` | Additional file extensions for module replacement |

### Source Formats

| Format | Example |
|--------|---------|
| GitHub shorthand | `user/repo` |
| Full URL | `github.com/user/repo` |
| Other Git hosts | `codeberg.org/user/repo` |
| Specific tag | `user/repo@v1.0.0` |
| Specific commit | `user/repo@abc1234` |
| Local directory | `./my-template` |

## Examples

Create a new project from a GitHub template:

```bash
gohatch user/go-template github.com/me/myapp
```

Use a specific version:

```bash
gohatch user/go-template@v1.0.0 github.com/me/myapp
```

Specify output directory:

```bash
gohatch user/go-template github.com/me/myapp ./projects/myapp
```

Also replace module path in config files:

```bash
gohatch -e toml -e yaml user/go-template github.com/me/myapp
```

Use a local template:

```bash
gohatch ./my-template github.com/me/myapp
```

## License

This project is licensed under the [European Union Public License 1.2](https://eupl.eu/) (EUPL-1.2).
