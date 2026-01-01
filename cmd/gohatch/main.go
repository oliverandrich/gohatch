// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/oliverandrich/gohatch/internal/rewrite"
	"codeberg.org/oliverandrich/gohatch/internal/source"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/urfave/cli/v3"
)

var version = "dev"

var (
	srcInput   string
	module     string
	directory  string
	extensions []string
	variables  []string
	dryRun     bool
	force      bool
	noGitInit  bool
	verbose    bool
)

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
				Destination: &extensions,
			},
			&cli.StringSliceFlag{
				Name:        "var",
				Aliases:     []string{"v"},
				Usage:       "set template variable (e.g., --var Author=\"Name\")",
				Destination: &variables,
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "show what would be done without making any changes",
				Destination: &dryRun,
			},
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "proceed even if template has no go.mod",
				Destination: &force,
			},
			&cli.BoolFlag{
				Name:        "no-git-init",
				Usage:       "skip git repository initialization",
				Destination: &noGitInit,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "show detailed progress output",
				Destination: &verbose,
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

	// Parse the source
	src, err := source.Parse(srcInput)
	if err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	// Dry-run mode: show what would be done
	if dryRun {
		return runDryRun(src)
	}

	return executeScaffold(ctx, src)
}

func executeScaffold(ctx context.Context, src source.Source) error {
	if err := validateDirectory(directory); err != nil {
		return err
	}

	if err := fetchTemplate(ctx, src); err != nil {
		return err
	}

	if err := validateGoMod(); err != nil {
		return err
	}

	vars := parseVariables(variables, path.Base(directory))

	if err := renamePaths(vars); err != nil {
		return err
	}

	if err := rewriteModule(); err != nil {
		return err
	}

	if err := replaceVariables(vars); err != nil {
		return err
	}

	if !noGitInit {
		if err := initGitRepo(directory); err != nil {
			return fmt.Errorf("initializing git repository: %w", err)
		}
	}

	fmt.Printf("Created %s\n", directory)
	return nil
}

func fetchTemplate(ctx context.Context, src source.Source) error {
	fmt.Printf("Fetching template from %s...\n", srcInput)
	if err := src.Fetch(ctx, directory); err != nil {
		return fmt.Errorf("fetching template: %w", err)
	}

	verboseLog("Removing template .git directory")
	if err := os.RemoveAll(filepath.Join(directory, ".git")); err != nil {
		return fmt.Errorf("removing template .git: %w", err)
	}

	return nil
}

func validateGoMod() error {
	if rewrite.HasGoMod(directory) {
		return nil
	}

	if !force {
		_ = os.RemoveAll(directory)
		return fmt.Errorf("template has no go.mod (use --force to proceed anyway)")
	}

	fmt.Println("Warning: template has no go.mod, skipping module rewrite")
	return nil
}

func renamePaths(vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	renamedPaths, err := rewrite.RenamePaths(directory, vars)
	if err != nil {
		return fmt.Errorf("renaming paths: %w", err)
	}

	if len(renamedPaths) > 0 {
		fmt.Println("Renaming paths...")
		for _, r := range renamedPaths {
			verboseLog("Renamed: %s", r)
		}
	}

	return nil
}

func rewriteModule() error {
	if !rewrite.HasGoMod(directory) {
		return nil
	}

	oldModule, err := rewrite.ReadModulePath(directory)
	if err != nil {
		return fmt.Errorf("reading module path: %w", err)
	}
	verboseLog("Found go.mod with module: %s", oldModule)

	if oldModule == module {
		return nil
	}

	fmt.Printf("Rewriting module %s → %s\n", oldModule, module)
	modifiedFiles, err := rewrite.Module(directory, module, extensions)
	if err != nil {
		return fmt.Errorf("rewriting module: %w", err)
	}

	for _, f := range modifiedFiles {
		verboseLog("Rewritten: %s", f)
	}

	return nil
}

func replaceVariables(vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	fmt.Printf("Replacing variables: %v\n", formatVariables(vars))
	modifiedFiles, err := rewrite.Variables(directory, vars, extensions)
	if err != nil {
		return fmt.Errorf("replacing variables: %w", err)
	}

	for _, f := range modifiedFiles {
		verboseLog("Replaced variables in: %s", f)
	}

	return nil
}

// parseVariables converts CLI key=value pairs to a map.
// Sets ProjectName to defaultProjectName if not overridden.
func parseVariables(vars []string, defaultProjectName string) map[string]string {
	result := map[string]string{
		"ProjectName": defaultProjectName,
	}
	for _, v := range vars {
		if key, value, ok := strings.Cut(v, "="); ok {
			result[key] = value
		}
	}
	return result
}

// formatVariables formats variables for display.
func formatVariables(vars map[string]string) string {
	parts := make([]string, 0, len(vars))
	for k, v := range vars {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}

// verboseLog prints a message only if verbose mode is enabled.
func verboseLog(format string, args ...any) {
	if verbose {
		fmt.Printf("  "+format+"\n", args...)
	}
}

func runDryRun(src source.Source) error {
	fmt.Println("Dry-run mode: no changes will be made")
	fmt.Println()

	// Show source info
	switch s := src.(type) {
	case *source.GitSource:
		fmt.Printf("Source:    %s\n", s.URL)
		if s.Version != "" {
			fmt.Printf("Version:   %s\n", s.Version)
		}
	case *source.LocalSource:
		fmt.Printf("Source:    %s (local)\n", s.Path)
	}

	// Show target info
	fmt.Printf("Directory: %s\n", directory)
	fmt.Printf("Module:    %s\n", module)

	// Show extensions if any
	if len(extensions) > 0 {
		fmt.Printf("Extensions: %v\n", extensions)
	}

	// Show variables
	vars := parseVariables(variables, path.Base(directory))
	fmt.Printf("Variables: %s\n", formatVariables(vars))

	// Show force flag
	if force {
		fmt.Println("Force:     --force (skip go.mod validation)")
	}

	// Show git init status
	if noGitInit {
		fmt.Println("Git:       --no-git-init (skip initialization)")
	}

	fmt.Println()
	fmt.Println("Would fetch template and rewrite module path in all .go files.")
	if len(extensions) > 0 {
		fmt.Println("Would also replace module path in files with specified extensions.")
	}
	fmt.Println("Would replace template variables (__Key__ → Value).")
	if !noGitInit {
		fmt.Println("Would initialize git repository with initial commit.")
	}

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

// initGitRepo initializes a git repository and creates an initial commit.
func initGitRepo(dir string) error {
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	// Add all files
	if err = worktree.AddGlob("."); err != nil {
		return fmt.Errorf("staging files: %w", err)
	}

	// Create initial commit
	_, err = worktree.Commit("Initial commit.", &git.CommitOptions{
		Author: getGitAuthor(),
	})
	if err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	fmt.Println("Initialized git repository with initial commit")
	return nil
}

// getGitAuthor reads the git author from the user's global git config.
// Falls back to gohatch defaults if not configured.
func getGitAuthor() *object.Signature {
	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil && cfg.User.Name != "" && cfg.User.Email != "" {
		return &object.Signature{
			Name:  cfg.User.Name,
			Email: cfg.User.Email,
			When:  time.Now(),
		}
	}
	return &object.Signature{
		Name:  "gohatch",
		Email: "gohatch@localhost",
		When:  time.Now(),
	}
}
