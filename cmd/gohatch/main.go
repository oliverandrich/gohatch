// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gohatchcfg "github.com/oliverandrich/gohatch/internal/config"
	"github.com/oliverandrich/gohatch/internal/rewrite"
	"github.com/oliverandrich/gohatch/internal/source"
	"github.com/urfave/cli/v3"
	gomodule "golang.org/x/mod/module"
	"golang.org/x/term"
)

// hookTimeout is the maximum duration for a single hook command.
const hookTimeout = 5 * time.Minute

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

func run(ctx context.Context, cmd *cli.Command) error {
	// Show help if required arguments are missing
	if opts.srcInput == "" || opts.module == "" {
		return cli.ShowAppHelp(cmd)
	}

	// Validate module path
	if err := gomodule.CheckPath(opts.module); err != nil {
		return fmt.Errorf("invalid module path %q: %w", opts.module, err)
	}

	// Default directory to last element of module path
	if opts.directory == "" {
		opts.directory = path.Base(opts.module)
	}

	// Parse the source
	src, err := source.Parse(opts.srcInput)
	if err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	// Dry-run mode: show what would be done
	if opts.dryRun {
		return runDryRun(ctx, src)
	}

	return executeScaffold(ctx, src)
}

func executeScaffold(ctx context.Context, src source.Source) error {
	if err := validateDirectory(opts.directory); err != nil {
		return err
	}

	if err := fetchTemplate(ctx, src); err != nil {
		return err
	}

	// From here on, cleanup directory on error
	scaffoldErr := scaffold(ctx)
	if scaffoldErr != nil {
		_ = os.RemoveAll(opts.directory)
		return scaffoldErr
	}

	if !opts.noGitInit {
		if err := initGitRepo(opts.directory); err != nil {
			return fmt.Errorf("initializing git repository: %w", err)
		}
	}

	fmt.Printf("\nCreated %s\n", opts.directory)
	return nil
}

func scaffold(ctx context.Context) error {
	cfg, mergedExtensions, err := loadConfig()
	if err != nil {
		return err
	}

	if err := validateGoMod(); err != nil {
		return err
	}

	vars := parseVariables(opts.variables, path.Base(opts.directory), opts.module)

	if err := detectUnsetVars(vars, mergedExtensions); err != nil {
		return err
	}

	if err := renamePaths(vars); err != nil {
		return err
	}

	if err := rewriteModule(mergedExtensions); err != nil {
		return err
	}

	if err := replaceVariables(vars, mergedExtensions); err != nil {
		return err
	}

	if err := removeConfigFile(); err != nil {
		return err
	}

	return runHooks(ctx, cfg.Hooks, opts.directory, vars)
}

func fetchTemplate(ctx context.Context, src source.Source) error {
	fmt.Printf("Fetching template from %s...\n", opts.srcInput)
	if err := src.Fetch(ctx, opts.directory); err != nil {
		return fmt.Errorf("fetching template: %w", err)
	}

	verboseLog("Removing template .git directory")
	if err := os.RemoveAll(filepath.Join(opts.directory, ".git")); err != nil {
		return fmt.Errorf("removing template .git: %w", err)
	}

	return nil
}

func removeConfigFile() error {
	if gohatchcfg.Exists(opts.directory) && !opts.keepConfig {
		if err := gohatchcfg.Remove(opts.directory); err != nil {
			return fmt.Errorf("removing config: %w", err)
		}
		verboseLog("Removed %s", gohatchcfg.ConfigFile)
	}
	return nil
}

func loadConfig() (*gohatchcfg.Config, []string, error) {
	cfg, err := gohatchcfg.Load(opts.directory)
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	if gohatchcfg.Exists(opts.directory) {
		verboseLog("Found %s", gohatchcfg.ConfigFile)
	}

	merged := mergeExtensions(opts.extensions, cfg.Extensions)
	if len(merged) > 0 {
		verboseLog("Extensions: %v", merged)
	}
	return cfg, merged, nil
}

func validateGoMod() error {
	if rewrite.HasGoMod(opts.directory) {
		return nil
	}

	if !opts.force {
		return fmt.Errorf("template has no go.mod (use --force to proceed anyway)")
	}

	fmt.Println("Warning: template has no go.mod, skipping module rewrite")
	return nil
}

func renamePaths(vars map[string]string) error {
	if len(vars) == 0 {
		return nil
	}

	renamedPaths, err := rewrite.RenamePaths(opts.directory, vars)
	if err != nil {
		return fmt.Errorf("renaming paths: %w", err)
	}

	if len(renamedPaths) > 0 {
		fmt.Println("\nRenaming paths...")
		for _, r := range renamedPaths {
			verboseLog("Renamed: %s", r)
		}
	}

	return nil
}

func rewriteModule(exts []string) error {
	if !rewrite.HasGoMod(opts.directory) {
		return nil
	}

	oldModule, err := rewrite.ReadModulePath(opts.directory)
	if err != nil {
		return fmt.Errorf("reading module path: %w", err)
	}
	verboseLog("Found go.mod with module: %s", oldModule)

	if oldModule == opts.module {
		return nil
	}

	fmt.Printf("\nRewriting module %s → %s\n", oldModule, opts.module)
	modifiedFiles, err := rewrite.Module(opts.directory, opts.module, exts)
	if err != nil {
		return fmt.Errorf("rewriting module: %w", err)
	}

	for _, f := range modifiedFiles {
		verboseLog("Rewritten: %s", f)
	}

	return nil
}

func replaceVariables(vars map[string]string, exts []string) error {
	if len(vars) == 0 {
		return nil
	}

	fmt.Printf("\nReplacing variables: %v\n", formatVariables(vars))
	modifiedFiles, err := rewrite.Variables(opts.directory, vars, exts)
	if err != nil {
		return fmt.Errorf("replacing variables: %w", err)
	}

	for _, f := range modifiedFiles {
		verboseLog("Replaced variables in: %s", f)
	}

	return nil
}

func detectUnsetVars(vars map[string]string, exts []string) error {
	unset, err := rewrite.DetectUnsetVars(opts.directory, vars, exts)
	if err != nil {
		return fmt.Errorf("detecting unset variables: %w", err)
	}

	if len(unset.InPaths) == 0 && len(unset.InContents) == 0 {
		return nil
	}

	// Prompt interactively if possible
	if !opts.noPrompt && isInteractive() {
		if promptErr := promptForVars(unset, vars); promptErr != nil {
			return fmt.Errorf("prompting for variables: %w", promptErr)
		}
		// Re-check after prompting: vars that got values are no longer unset
		unset = recomputeUnset(unset, vars)
	}

	// Unset variables in paths are always an error
	if len(unset.InPaths) > 0 {
		return fmt.Errorf("unset template variables in paths: %v (set them with --var Key=Value)", unset.InPaths)
	}

	// Unset variables in contents: error in strict mode, warning otherwise
	if len(unset.InContents) > 0 {
		if opts.strict {
			return fmt.Errorf("unset template variables in file contents (--strict mode): %v (set them with --var Key=Value)", unset.InContents)
		}
		for _, name := range unset.InContents {
			fmt.Printf("Warning: unset template variable __%s__ will be removed from file contents\n", name)
			vars[name] = ""
		}
	}

	return nil
}

// isInteractive returns true if stdin is a terminal.
// Extracted as a variable for testing.
var isInteractive = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// promptForVars prompts the user for each unset variable using an interactive form.
func promptForVars(unset *rewrite.UnsetVars, vars map[string]string) error {
	// Collect unique variable names, path vars first (required)
	seen := make(map[string]bool)
	var pathVars, contentVars []string
	for _, name := range unset.InPaths {
		if !seen[name] {
			seen[name] = true
			pathVars = append(pathVars, name)
		}
	}
	for _, name := range unset.InContents {
		if !seen[name] {
			seen[name] = true
			contentVars = append(contentVars, name)
		}
	}

	// Use local string pointers so huh can bind to them
	type varBinding struct {
		name string
		val  string
	}
	bindings := make([]*varBinding, 0, len(pathVars)+len(contentVars))
	fields := make([]huh.Field, 0, len(pathVars)+len(contentVars))

	for _, name := range pathVars {
		b := &varBinding{name: name}
		bindings = append(bindings, b)
		fields = append(fields, huh.NewInput().
			Title(name+" (required, used in paths)").
			Value(&b.val).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("%s is required (used in file paths)", b.name)
				}
				return nil
			}))
	}

	for _, name := range contentVars {
		b := &varBinding{name: name}
		bindings = append(bindings, b)
		fields = append(fields, huh.NewInput().
			Title(name).
			Value(&b.val))
	}

	form := huh.NewForm(huh.NewGroup(fields...))

	fmt.Println()
	if err := form.Run(); err != nil {
		return err
	}

	// Copy prompted values back to vars map
	for _, b := range bindings {
		vars[b.name] = b.val
	}
	return nil
}

// recomputeUnset filters out variables that now have values after prompting.
func recomputeUnset(unset *rewrite.UnsetVars, vars map[string]string) *rewrite.UnsetVars {
	result := &rewrite.UnsetVars{}
	for _, name := range unset.InPaths {
		if v, ok := vars[name]; !ok || v == "" {
			result.InPaths = append(result.InPaths, name)
		}
	}
	for _, name := range unset.InContents {
		if _, ok := vars[name]; !ok {
			result.InContents = append(result.InContents, name)
		}
	}
	sort.Strings(result.InPaths)
	sort.Strings(result.InContents)
	return result
}

// parseVariables converts CLI key=value pairs to a map.
// Sets ProjectName and GitUser automatically from module path if not overridden.
func parseVariables(vars []string, defaultProjectName string, modulePath string) map[string]string {
	result := map[string]string{
		"ProjectName": defaultProjectName,
	}

	// Extract GitUser from module path (e.g., github.com/user/repo -> user)
	if parts := strings.Split(modulePath, "/"); len(parts) >= 2 {
		result["GitUser"] = parts[1]
	}

	// Override with CLI variables
	for _, v := range vars {
		if key, value, ok := strings.Cut(v, "="); ok {
			result[key] = value
		}
	}
	return result
}

// formatVariables formats variables for display in deterministic order.
func formatVariables(vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(vars))
	for _, k := range keys {
		parts = append(parts, k+"="+vars[k])
	}
	return strings.Join(parts, ", ")
}

// mergeExtensions combines CLI extensions with config extensions.
// CLI extensions are added to config extensions (union).
func mergeExtensions(cliExts, configExts []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(cliExts)+len(configExts))

	// Config extensions first
	for _, ext := range configExts {
		if !seen[ext] {
			seen[ext] = true
			result = append(result, ext)
		}
	}

	// CLI extensions added (if not already present)
	for _, ext := range cliExts {
		if !seen[ext] {
			seen[ext] = true
			result = append(result, ext)
		}
	}

	return result
}

// verboseLog prints a message only if verbose mode is enabled.
func verboseLog(format string, args ...any) {
	if opts.verbose {
		fmt.Printf("  "+format+"\n", args...)
	}
}

func runDryRun(ctx context.Context, src source.Source) error {
	fmt.Println("Dry-run mode: no changes will be made")
	fmt.Println()

	// Fetch template into temp directory
	tmpDir, err := os.MkdirTemp("", "gohatch-dry-run-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := src.Fetch(ctx, tmpDir); err != nil {
		return fmt.Errorf("fetching template: %w", err)
	}
	_ = os.RemoveAll(filepath.Join(tmpDir, ".git"))

	// Show source info
	printSourceInfo(src)
	fmt.Printf("Directory: %s\n", opts.directory)

	// Load config and merge extensions
	cfg, cfgErr := gohatchcfg.Load(tmpDir)
	mergedExt := opts.extensions
	if cfgErr == nil {
		mergedExt = mergeExtensions(opts.extensions, cfg.Extensions)
	}

	if len(mergedExt) > 0 {
		fmt.Printf("Extensions: %v\n", mergedExt)
	}

	// Module rewrite info
	printModuleInfo(tmpDir)

	// Variables
	vars := parseVariables(opts.variables, path.Base(opts.directory), opts.module)
	fmt.Println()
	printFileTree(tmpDir)
	printPathRenames(tmpDir, vars)
	printVariables(vars)
	printUnsetVarWarnings(tmpDir, vars, mergedExt)

	// Hooks
	if cfgErr == nil {
		printHooks(cfg.Hooks, vars)
	}

	// Flags
	fmt.Println()
	printFlags()

	return nil
}

func printSourceInfo(src source.Source) {
	switch s := src.(type) {
	case *source.GitSource:
		if s.Version != "" {
			fmt.Printf("Source:     %s (%s)\n", s.URL, s.Version)
		} else {
			fmt.Printf("Source:     %s\n", s.URL)
		}
	case *source.LocalSource:
		fmt.Printf("Source:     %s (local)\n", s.Path)
	}
}

func printModuleInfo(tmpDir string) {
	if !rewrite.HasGoMod(tmpDir) {
		fmt.Println("Warning:   no go.mod found in template")
		return
	}
	oldModule, err := rewrite.ReadModulePath(tmpDir)
	if err != nil {
		return
	}
	fmt.Printf("Module:    %s → %s\n", oldModule, opts.module)
}

func printFileTree(dir string) {
	fmt.Println("Files:")
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			fmt.Printf("  %s/\n", rel)
		} else {
			fmt.Printf("  %s\n", rel)
		}
		return nil
	})
}

func printPathRenames(dir string, vars map[string]string) {
	if len(vars) == 0 {
		return
	}

	var renames []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		name := d.Name()
		newName := name
		for key, value := range vars {
			placeholder := "__" + key + "__"
			newName = strings.ReplaceAll(newName, placeholder, value)
		}
		if newName != name {
			rel, _ := filepath.Rel(dir, p)
			newRel := filepath.Join(filepath.Dir(rel), newName)
			renames = append(renames, fmt.Sprintf("  %s → %s", rel, newRel))
		}
		return nil
	})

	if len(renames) > 0 {
		fmt.Println()
		fmt.Println("Renamed paths:")
		for _, r := range renames {
			fmt.Println(r)
		}
	}
}

func printVariables(vars map[string]string) {
	if len(vars) == 0 {
		return
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println()
	fmt.Println("Variables:")
	for _, k := range keys {
		fmt.Printf("  %-15s = %s\n", k, vars[k])
	}
}

func printUnsetVarWarnings(tmpDir string, vars map[string]string, exts []string) {
	unset, err := rewrite.DetectUnsetVars(tmpDir, vars, exts)
	if err != nil {
		fmt.Printf("\nWarning: could not detect unset variables: %v\n", err)
		return
	}
	for _, name := range unset.InPaths {
		fmt.Printf("\nWarning: unset template variable __%s__ in path (must be set with --var %s=Value)\n", name, name)
	}
	for _, name := range unset.InContents {
		fmt.Printf("\nWarning: unset template variable __%s__ will be removed from file contents\n", name)
	}
}

func printFlags() {
	if opts.force {
		fmt.Println("Would skip go.mod validation (--force).")
	}
	if opts.strict {
		fmt.Println("Would treat unset variables in file contents as errors (--strict).")
	}
	if !opts.keepConfig && gohatchcfg.ConfigFile != "" {
		fmt.Println("Would remove .gohatch.toml from output.")
	}
	if opts.noHooks {
		fmt.Println("Would skip hooks (--no-hooks).")
	}
	if !opts.noGitInit {
		fmt.Println("Would initialize git repository with initial commit.")
	}
}

// substituteHookVars replaces __VarName__ placeholders in a hook command.
func substituteHookVars(command string, vars map[string]string) string {
	for key, value := range vars {
		command = strings.ReplaceAll(command, "__"+key+"__", value)
	}
	return command
}

// confirmHooks asks the user to confirm hook execution.
// Defined as a variable for test overriding.
var confirmHooks = func(hooks []gohatchcfg.Hook, vars map[string]string) (bool, error) {
	fmt.Println("\nPost-generation hooks:")
	for i, h := range hooks {
		resolved := substituteHookVars(h.Command, vars)
		fmt.Printf("  %d. %s: %s\n", i+1, h.Name, resolved)
	}
	fmt.Println()

	var confirmed bool
	form := huh.NewConfirm().Title("Run these hooks?").Value(&confirmed)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

// runHooks executes post-generation hooks sequentially.
func runHooks(ctx context.Context, hooks []gohatchcfg.Hook, dir string, vars map[string]string) error {
	if len(hooks) == 0 || opts.noHooks {
		return nil
	}

	if !isInteractive() {
		fmt.Println("Warning: skipping hooks (non-interactive terminal)")
		return nil
	}

	confirmed, err := confirmHooks(hooks, vars)
	if err != nil {
		return fmt.Errorf("confirming hooks: %w", err)
	}
	if !confirmed {
		fmt.Println("Skipping hooks")
		return nil
	}

	for _, h := range hooks {
		resolved := substituteHookVars(h.Command, vars)
		fmt.Printf("Running hook: %s\n", h.Name)
		hookCtx, cancel := context.WithTimeout(ctx, hookTimeout)
		cmd := exec.CommandContext(hookCtx, "sh", "-c", resolved) //nolint:gosec // hooks are user-confirmed
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		cancel()
		if err != nil {
			return fmt.Errorf("hook %q failed: %w", h.Name, err)
		}
	}

	return nil
}

// printHooks shows hook commands in dry-run output.
func printHooks(hooks []gohatchcfg.Hook, vars map[string]string) {
	if len(hooks) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("Hooks:")
	for i, h := range hooks {
		resolved := substituteHookVars(h.Command, vars)
		fmt.Printf("  %d. %s: %s\n", i+1, h.Name, resolved)
	}
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
	repo, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName("main"),
		},
	})
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

	fmt.Println("\nInitialized git repository with initial commit")
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
