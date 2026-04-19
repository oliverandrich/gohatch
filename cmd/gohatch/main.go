// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v3"
)

var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

type options struct {
	srcInput   string
	module     string
	directory  string
	extensions []string
	variables  []string
	dryRun     bool
	force      bool
	noGitInit  bool
	keepConfig bool
	verbose    bool
	strict     bool
	noPrompt   bool
	noHooks    bool
}

var opts options

func main() {
	// Remove -v alias from version flag to avoid conflict with --var
	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "print the version",
	}

	cmd := &cli.Command{
		Name:      "gohatch",
		Usage:     "A project scaffolding tool for Go",
		Version:   version,
		ArgsUsage: "<source> <module> [directory]",
		Description: `Create a new Go project from a template.

Source formats:
  user/repo                     GitHub shorthand
  github.com/user/repo          Full URL
  codeberg.org/user/repo        Other Git hosts
  user/repo@v1.0.0              Specific tag
  user/repo@main                Specific branch
  user/repo@abc1234             Specific commit
  ./local-template              Local directory

Examples:
  gohatch user/template github.com/me/myapp
  gohatch github.com/user/template@v1.0.0 github.com/me/myapp
  gohatch user/template@main github.com/me/myapp
  gohatch ./local-template github.com/me/myapp customdir
  gohatch -e toml -e justfile user/template github.com/me/myapp
  gohatch --var Author="Your Name" user/template github.com/me/myapp
  gohatch --dry-run user/template github.com/me/myapp
  gohatch --force user/non-go-template github.com/me/myapp`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "extension",
				Aliases:     []string{"e"},
				Usage:       "additional file extensions or filenames for replacement (e.g., -e toml -e justfile)",
				Destination: &opts.extensions,
			},
			&cli.StringSliceFlag{
				Name:        "var",
				Aliases:     []string{"v"},
				Usage:       "set template variable (e.g., --var Author=\"Name\")",
				Destination: &opts.variables,
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "show what would be done without making any changes",
				Destination: &opts.dryRun,
			},
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "proceed even if template has no go.mod",
				Destination: &opts.force,
			},
			&cli.BoolFlag{
				Name:        "no-git-init",
				Usage:       "skip git repository initialization",
				Destination: &opts.noGitInit,
			},
			&cli.BoolFlag{
				Name:        "keep-config",
				Usage:       "keep .gohatch.toml config file in output",
				Destination: &opts.keepConfig,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "show detailed progress output",
				Destination: &opts.verbose,
			},
			&cli.BoolFlag{
				Name:        "strict",
				Usage:       "treat unset template variables in file contents as errors",
				Destination: &opts.strict,
			},
			&cli.BoolFlag{
				Name:        "no-prompt",
				Usage:       "disable interactive prompting for missing variables",
				Destination: &opts.noPrompt,
			},
			&cli.BoolFlag{
				Name:        "no-hooks",
				Usage:       "skip post-generation hooks defined in .gohatch.toml",
				Destination: &opts.noHooks,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "source",
				UsageText:   "template source (URL, shorthand, or local path)",
				Destination: &opts.srcInput,
			},
			&cli.StringArg{
				Name:        "module",
				UsageText:   "new module path",
				Destination: &opts.module,
			},
			&cli.StringArg{
				Name:        "directory",
				UsageText:   "output directory (optional)",
				Destination: &opts.directory,
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
