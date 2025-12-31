// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/oliverandrich/gohatch/internal/rewrite"
	"github.com/oliverandrich/gohatch/internal/source"
	"github.com/urfave/cli/v3"
)

var version = "dev"

var (
	srcInput   string
	module     string
	directory  string
	extensions []string
)

func main() {
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
  user/repo@abc1234             Specific commit
  ./local-template              Local directory

Examples:
  gohatch user/template github.com/me/myapp
  gohatch github.com/user/template@v1.0.0 github.com/me/myapp
  gohatch ./local-template github.com/me/myapp customdir
  gohatch -e toml -e sh user/template github.com/me/myapp`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "extension",
				Aliases:     []string{"e"},
				Usage:       "additional file extensions for module replacement (e.g., -e toml -e sh)",
				Destination: &extensions,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "source",
				UsageText:   "template source (URL, shorthand, or local path)",
				Destination: &srcInput,
			},
			&cli.StringArg{
				Name:        "module",
				UsageText:   "new module path",
				Destination: &module,
			},
			&cli.StringArg{
				Name:        "directory",
				UsageText:   "output directory (optional)",
				Destination: &directory,
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	// Show help if required arguments are missing
	if srcInput == "" || module == "" {
		return cli.ShowAppHelp(cmd)
	}

	// Default directory to last element of module path
	if directory == "" {
		directory = path.Base(module)
	}

	// Validate target directory
	if err := validateDirectory(directory); err != nil {
		return err
	}

	// Parse the source
	src, err := source.Parse(srcInput)
	if err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	// Fetch the template
	fmt.Printf("Fetching template from %s...\n", srcInput)
	if err := src.Fetch(ctx, directory); err != nil {
		return fmt.Errorf("fetching template: %w", err)
	}

	// Rewrite module path if go.mod exists
	if rewrite.HasGoMod(directory) {
		oldModule, err := rewrite.ReadModulePath(directory)
		if err != nil {
			return fmt.Errorf("reading module path: %w", err)
		}

		if oldModule != module {
			fmt.Printf("Rewriting module %s → %s\n", oldModule, module)
			if err := rewrite.Module(directory, module, extensions); err != nil {
				return fmt.Errorf("rewriting module: %w", err)
			}
		}
	}

	fmt.Printf("Created %s\n", directory)
	return nil
}

// validateDirectory checks that the target directory doesn't exist or is empty.
func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil // Directory doesn't exist, OK
	}
	if err != nil {
		return fmt.Errorf("checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", dir)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("directory %s is not empty", dir)
	}

	return nil
}
