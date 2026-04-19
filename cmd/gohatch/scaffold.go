// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/huh/v2"
	gohatchcfg "github.com/oliverandrich/gohatch/internal/config"
	"github.com/oliverandrich/gohatch/internal/rewrite"
	"github.com/oliverandrich/gohatch/internal/source"
	"github.com/urfave/cli/v3"
	gomodule "golang.org/x/mod/module"
	"golang.org/x/term"
)

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
