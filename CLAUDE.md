# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gohatch is a project scaffolding tool for Go, similar to gonew but with additional features.

- Module: `github.com/oliverandrich/gohatch`
- License: EUPL-1.2

## Project Structure

```
cmd/gohatch/    # CLI entry point
internal/       # Internal packages
```

## Development Commands

```bash
just build          # Build binary to build/gohatch
just test           # Run tests
just test -v        # Run tests with verbose output
just fmt            # Format code
just lint           # Run golangci-lint
just check          # Run fmt, lint, and test
just clean          # Remove build artifacts
just install        # Install to $GOPATH/bin
```

## License Headers

All new `.go` files must start with:

```go
// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich
```

## Git

- Use Conventional Commits (feat, fix, docs, refactor, test, chore, perf)
- Never commit autonomously - wait for explicit instruction
- No AI tool references in commits

## Tools

- Use `fd` instead of `find`
- Use `rg` instead of `grep`
