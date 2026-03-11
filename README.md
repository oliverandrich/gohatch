# gohatch

<a href="https://github.com/oliverandrich/gohatch/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/oliverandrich/gohatch/ci.yml?branch=main&label=CI&style=for-the-badge" alt="CI"></a>
<a href="https://github.com/oliverandrich/gohatch/releases"><img src="https://img.shields.io/github/v/release/oliverandrich/gohatch?style=for-the-badge" alt="Release"></a>
<a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/oliverandrich/gohatch?style=for-the-badge" alt="Go Version"></a>
<a href="https://goreportcard.com/report/github.com/oliverandrich/gohatch"><img src="https://goreportcard.com/badge/github.com/oliverandrich/gohatch?style=for-the-badge" alt="Go Report Card"></a>
<a href="/LICENSE"><img src="https://img.shields.io/github/license/oliverandrich/gohatch?style=for-the-badge" alt="License"></a>

A project scaffolding tool for Go, inspired by [gonew](https://go.dev/blog/gonew) with additional features.

- Clone templates from any Git host or local directories
- Automatic module path rewriting in `go.mod` and all `.go` files
- Template variable substitution with interactive prompting
- Post-generation hooks via `.gohatch.toml`

## Installation

```bash
brew install oliverandrich/tap/gohatch
```

```bash
go install github.com/oliverandrich/gohatch/cmd/gohatch@latest
```

## Quick Start

```bash
gohatch user/go-template github.com/me/myapp
```

## Documentation

Full documentation is available at [gohatch.someonewho.codes](https://gohatch.someonewho.codes/).

## License

[European Union Public License 1.2](https://eupl.eu/) (EUPL-1.2)
