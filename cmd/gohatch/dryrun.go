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

	gohatchcfg "github.com/oliverandrich/gohatch/internal/config"
	"github.com/oliverandrich/gohatch/internal/rewrite"
	"github.com/oliverandrich/gohatch/internal/source"
)

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
