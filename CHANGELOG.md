# Changelog

All notable changes to gohatch are documented here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

Earlier releases (≤ v0.7.1) are only covered by the git history.

## 0.8.0 — 2026-04-19

### Added

- **Interactive variable prompting** — when a template uses `__VarName__` placeholders that are not supplied via `--var` on the CLI, gohatch now asks for each missing value via an interactive form (powered by [huh](https://github.com/charmbracelet/huh)). Path variables are required; content variables are optional. Disable with `--no-prompt` or a non-interactive TTY.
- **Post-generation hooks** — templates can declare shell commands under a `[[hooks]]` array in `.gohatch.toml` to run after scaffolding. Hooks execute with `__VarName__` placeholders substituted, are confirmed interactively before running, and time out after 5 minutes each. Disable with `--no-hooks` or a non-interactive TTY.
- **Concrete dry-run preview** — `--dry-run` now fetches the template into a temp directory and shows the exact file tree, path renames, variable substitutions, unset-variable warnings, and hook commands that would execute — no more hand-waved plan.
- **Documentation site** — MkDocs + Material theme docs live at [gohatch.readthedocs.io](https://gohatch.readthedocs.io/). Covers installation, quick start, usage, template authoring, and `.gohatch.toml` reference.

### Changed

- **CI and release workflows hardened with [zizmor](https://docs.zizmor.sh/)** — every `actions/*` and `golangci/*` reference is now pinned to a commit SHA with a version comment, `persist-credentials: false` on each checkout, and `setup-go`'s built-in cache disabled to close the tag-push poisoning path. A new `zizmor` CI job audits all workflow files on every PR and push.
- **`cmd/gohatch/main.go` split into focused sibling files** — the 915-LOC `main.go` is now 154 LOC of pure CLI wiring; scaffold orchestration (`scaffold.go`), dry-run printing (`dryrun.go`), hook execution (`hooks.go`), and git init (`git.go`) live in their own files within the same `main` package. No API changes — purely mechanical cleanup.
- **`rewrite` package coverage raised** from 87.3 % to 89.2 % with two edge-case tests that exercise the binary-file short-circuit (skipping files with NUL bytes even if they match an extra-extension pattern).
- **Global state consolidated into an `options` struct** — loose `var x` declarations were gathered into a single `options` struct, simplifying test setup and making the CLI flag surface visible in one place.

### Fixed

- **`github.com/go-git/go-git` bumped to v5.18.0** — pulled in automatically by `go mod tidy` during a refactor; resolves [GO-2026-4909](https://pkg.go.dev/vuln/GO-2026-4909) (panic in Index v4 decoding) and [GO-2026-4910](https://pkg.go.dev/vuln/GO-2026-4910) (asymmetric memory consumption via a malicious idx file). `govulncheck` now reports no findings.
