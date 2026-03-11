# CLI Reference

## Synopsis

```
gohatch [options] <source> <module> [directory]
```

## Arguments

| Argument    | Description                                                     |
| ----------- | --------------------------------------------------------------- |
| `source`    | Template source (see [source formats](#source-formats) below)   |
| `module`    | New Go module path                                              |
| `directory` | Output directory (optional, defaults to last element of module) |

## Options

| Flag              | Description                                                    |
| ----------------- | -------------------------------------------------------------- |
| `-e, --extension` | Additional file extensions or filenames for replacement         |
| `-v, --var`       | Set template variable (e.g., `--var Author="Name"`)            |
| `--dry-run`       | Show what would be done without making any changes             |
| `-f, --force`     | Proceed even if template has no `go.mod`                       |
| `--no-git-init`   | Skip git repository initialization                             |
| `--keep-config`   | Keep `.gohatch.toml` config file in output                     |
| `--verbose`       | Show detailed progress output                                  |
| `--strict`        | Treat unset template variables in file contents as errors      |
| `--no-prompt`     | Disable interactive prompting for missing variables            |
| `--no-hooks`      | Skip post-generation hooks defined in `.gohatch.toml`          |
| `--version`       | Print the version                                              |

## Source Formats

| Format           | Example                  |
| ---------------- | ------------------------ |
| GitHub shorthand | `user/repo`              |
| Full URL         | `github.com/user/repo`   |
| Other Git hosts  | `codeberg.org/user/repo` |
| Specific tag     | `user/repo@v1.0.0`       |
| Specific branch  | `user/repo@main`         |
| Specific commit  | `user/repo@abc1234`      |
| Local directory  | `./my-template`          |

gohatch automatically detects whether the version is a tag, branch, or commit hash by querying the remote repository.

## Examples

### Basic usage

```bash
gohatch user/go-template github.com/me/myapp
```

### Use a specific tag

```bash
gohatch user/go-template@v1.0.0 github.com/me/myapp
```

### Use a specific branch

```bash
gohatch user/go-template@main github.com/me/myapp
```

### Specify output directory

```bash
gohatch user/go-template github.com/me/myapp ./projects/myapp
```

### Replace in additional file types

```bash
gohatch -e toml -e yaml user/go-template github.com/me/myapp
```

### Replace in specific files by name

```bash
gohatch -e yml -e justfile -e Makefile user/go-template github.com/me/myapp
```

### Use a local template

```bash
gohatch ./my-template github.com/me/myapp
```

### Set template variables

```bash
gohatch --var Author="Your Name" user/go-template github.com/me/myapp
```

### Multiple variables

```bash
gohatch -v ProjectName=MyApp -v Author="Your Name" user/go-template github.com/me/myapp
```

### Use a non-Go template

```bash
gohatch --force user/non-go-template github.com/me/myapp
```

### Strict mode (fail on unset variables)

```bash
gohatch --strict user/go-template github.com/me/myapp
```

### Skip interactive prompting

```bash
gohatch --no-prompt user/go-template github.com/me/myapp
```

### Skip hooks

```bash
gohatch --no-hooks user/go-template github.com/me/myapp
```

## Dry-Run Mode

Use `--dry-run` to preview what gohatch would do without making any changes:

```bash
gohatch --dry-run user/go-template github.com/me/myapp
```

Dry-run mode shows:

- Source information and target directory
- Template file tree
- Module path rewrite (`old → new`)
- Path renames (directories/files containing variable placeholders)
- Variable values (defaults and user-provided)
- Warnings about unset variables
- Hooks that would be executed
- Active flags (`--force`, `--strict`, `--no-hooks`, etc.)

The template is fetched into a temporary directory and cleaned up afterwards — no files are created in your working directory.

## Verbose Mode

Use `--verbose` to see detailed progress output during scaffolding:

```bash
gohatch --verbose user/go-template github.com/me/myapp
```

Verbose mode logs each individual file rewritten, path renamed, and variable replaced.
