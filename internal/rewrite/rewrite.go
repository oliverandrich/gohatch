// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// Module rewrites the module path in the given directory.
// It updates go.mod, all import paths in .go files, and performs
// string replacement in files with the specified extra extensions.
func Module(dir, newModule string, extraExtensions []string) error {
	// Read and parse go.mod
	goModPath := filepath.Clean(filepath.Join(dir, "go.mod"))
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.ParseLax(goModPath, data, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	oldModule := f.Module.Mod.Path
	if oldModule == newModule {
		return nil // Nothing to do
	}

	// Update go.mod
	err = f.AddModuleStmt(newModule)
	if err != nil {
		return fmt.Errorf("updating module statement: %w", err)
	}

	newData, err := f.Format()
	if err != nil {
		return fmt.Errorf("formatting go.mod: %w", err)
	}

	err = os.WriteFile(goModPath, newData, 0o600)
	if err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	// Rewrite imports in all .go files
	err = rewriteGoFiles(dir, oldModule, newModule)
	if err != nil {
		return fmt.Errorf("rewriting imports: %w", err)
	}

	// Rewrite extra extension files with simple string replacement
	if len(extraExtensions) > 0 {
		err = rewriteExtraFiles(dir, oldModule, newModule, extraExtensions)
		if err != nil {
			return fmt.Errorf("rewriting extra files: %w", err)
		}
	}

	return nil
}

// rewriteGoFiles walks through all .go files and rewrites import paths.
func rewriteGoFiles(dir, oldModule, newModule string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip vendor directory
			if d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		return rewriteGoFile(path, oldModule, newModule)
	})
}

// rewriteGoFile rewrites import paths in a single .go file using AST.
func rewriteGoFile(filePath, oldModule, newModule string) error {
	cleanPath := filepath.Clean(filePath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, cleanPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", cleanPath, err)
	}

	modified := false

	// Rewrite imports
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		if importPath == oldModule || strings.HasPrefix(importPath, oldModule+"/") {
			newPath := newModule + strings.TrimPrefix(importPath, oldModule)
			imp.Path.Value = `"` + newPath + `"`
			modified = true
		}
	}

	if !modified {
		return nil
	}

	// Format and write back
	var buf bytes.Buffer
	err = format.Node(&buf, fset, f)
	if err != nil {
		return fmt.Errorf("formatting %s: %w", cleanPath, err)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	return os.WriteFile(cleanPath, buf.Bytes(), info.Mode())
}

// rewriteExtraFiles walks through files with specified extensions
// and performs simple string replacement.
func rewriteExtraFiles(dir, oldModule, newModule string, extensions []string) error {
	// Normalize extensions (ensure they start with a dot)
	extSet := make(map[string]bool)
	for _, ext := range extensions {
		ext = strings.TrimPrefix(ext, ".")
		extSet["."+ext] = true
	}

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file extension matches
		ext := filepath.Ext(path)
		if !extSet[ext] {
			return nil
		}

		return rewriteTextFile(path, oldModule, newModule)
	})
}

// rewriteTextFile performs simple string replacement in a text file.
func rewriteTextFile(filePath, oldModule, newModule string) error {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cleanPath, err)
	}

	// Simple string replacement
	newData := bytes.ReplaceAll(data, []byte(oldModule), []byte(newModule))

	// Only write if changed
	if bytes.Equal(data, newData) {
		return nil
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	return os.WriteFile(cleanPath, newData, info.Mode())
}

// ReadModulePath reads the module path from a go.mod file.
func ReadModulePath(dir string) (string, error) {
	goModPath := filepath.Clean(filepath.Join(dir, "go.mod"))
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.ParseLax(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("parsing go.mod: %w", err)
	}

	return f.Module.Mod.Path, nil
}

// HasGoMod checks if the directory contains a go.mod file.
func HasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// Variables replaces template variables in all files.
// Variables use dunder-style syntax: __VariableName__ → value.
func Variables(dir string, vars map[string]string, extraExtensions []string) error {
	if len(vars) == 0 {
		return nil
	}

	// Build extension set: .go + extra extensions
	extSet := make(map[string]bool)
	extSet[".go"] = true
	for _, ext := range extraExtensions {
		ext = strings.TrimPrefix(ext, ".")
		extSet["."+ext] = true
	}

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file extension matches
		ext := filepath.Ext(path)
		if !extSet[ext] {
			return nil
		}

		return replaceVariablesInFile(path, vars)
	})
}

// replaceVariablesInFile replaces __Key__ with Value for all variables.
func replaceVariablesInFile(filePath string, vars map[string]string) error {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cleanPath, err)
	}

	// Replace each variable
	newData := data
	for key, value := range vars {
		placeholder := "__" + key + "__"
		newData = bytes.ReplaceAll(newData, []byte(placeholder), []byte(value))
	}

	// Only write if changed
	if bytes.Equal(data, newData) {
		return nil
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	return os.WriteFile(cleanPath, newData, info.Mode())
}
