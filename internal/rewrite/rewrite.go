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
// Returns the list of modified files.
func Module(dir, newModule string, extraExtensions []string) ([]string, error) {
	var modifiedFiles []string

	// Read and parse go.mod
	goModPath := filepath.Clean(filepath.Join(dir, "go.mod"))
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.ParseLax(goModPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parsing go.mod: %w", err)
	}

	oldModule := f.Module.Mod.Path
	if oldModule == newModule {
		return nil, nil // Nothing to do
	}

	// Update go.mod
	err = f.AddModuleStmt(newModule)
	if err != nil {
		return nil, fmt.Errorf("updating module statement: %w", err)
	}

	newData, err := f.Format()
	if err != nil {
		return nil, fmt.Errorf("formatting go.mod: %w", err)
	}

	err = os.WriteFile(goModPath, newData, 0o600)
	if err != nil {
		return nil, fmt.Errorf("writing go.mod: %w", err)
	}
	modifiedFiles = append(modifiedFiles, "go.mod")

	// Rewrite imports in all .go files
	goFiles, err := rewriteGoFiles(dir, oldModule, newModule)
	if err != nil {
		return nil, fmt.Errorf("rewriting imports: %w", err)
	}
	modifiedFiles = append(modifiedFiles, goFiles...)

	// Rewrite extra extension files with simple string replacement
	if len(extraExtensions) > 0 {
		extraFiles, err := rewriteExtraFiles(dir, oldModule, newModule, extraExtensions)
		if err != nil {
			return nil, fmt.Errorf("rewriting extra files: %w", err)
		}
		modifiedFiles = append(modifiedFiles, extraFiles...)
	}

	return modifiedFiles, nil
}

// rewriteGoFiles walks through all .go files and rewrites import paths.
// Returns the list of modified files.
func rewriteGoFiles(dir, oldModule, newModule string) ([]string, error) {
	var modifiedFiles []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

		modified, err := rewriteGoFile(path, oldModule, newModule)
		if err != nil {
			return err
		}
		if modified {
			relPath, _ := filepath.Rel(dir, path)
			modifiedFiles = append(modifiedFiles, relPath)
		}
		return nil
	})

	return modifiedFiles, err
}

// rewriteGoFile rewrites import paths in a single .go file using AST.
// Returns true if the file was modified.
func rewriteGoFile(filePath, oldModule, newModule string) (bool, error) {
	cleanPath := filepath.Clean(filePath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, cleanPath, nil, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %w", cleanPath, err)
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
		return false, nil
	}

	// Format and write back
	var buf bytes.Buffer
	err = format.Node(&buf, fset, f)
	if err != nil {
		return false, fmt.Errorf("formatting %s: %w", cleanPath, err)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return false, err
	}

	return true, os.WriteFile(cleanPath, buf.Bytes(), info.Mode())
}

// rewriteExtraFiles walks through files with specified extensions
// and performs simple string replacement.
// Returns the list of modified files.
func rewriteExtraFiles(dir, oldModule, newModule string, extensions []string) ([]string, error) {
	var modifiedFiles []string

	// Normalize extensions (ensure they start with a dot)
	extSet := make(map[string]bool)
	for _, ext := range extensions {
		ext = strings.TrimPrefix(ext, ".")
		extSet["."+ext] = true
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

		modified, err := rewriteTextFile(path, oldModule, newModule)
		if err != nil {
			return err
		}
		if modified {
			relPath, _ := filepath.Rel(dir, path)
			modifiedFiles = append(modifiedFiles, relPath)
		}
		return nil
	})

	return modifiedFiles, err
}

// rewriteTextFile performs simple string replacement in a text file.
// Returns true if the file was modified.
func rewriteTextFile(filePath, oldModule, newModule string) (bool, error) {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", cleanPath, err)
	}

	// Simple string replacement
	newData := bytes.ReplaceAll(data, []byte(oldModule), []byte(newModule))

	// Only write if changed
	if bytes.Equal(data, newData) {
		return false, nil
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return false, err
	}

	return true, os.WriteFile(cleanPath, newData, info.Mode())
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
// Returns the list of modified files.
func Variables(dir string, vars map[string]string, extraExtensions []string) ([]string, error) {
	if len(vars) == 0 {
		return nil, nil
	}

	var modifiedFiles []string

	// Build extension set: .go + extra extensions
	extSet := make(map[string]bool)
	extSet[".go"] = true
	for _, ext := range extraExtensions {
		ext = strings.TrimPrefix(ext, ".")
		extSet["."+ext] = true
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

		modified, err := replaceVariablesInFile(path, vars)
		if err != nil {
			return err
		}
		if modified {
			relPath, _ := filepath.Rel(dir, path)
			modifiedFiles = append(modifiedFiles, relPath)
		}
		return nil
	})

	return modifiedFiles, err
}

// replaceVariablesInFile replaces __Key__ with Value for all variables.
// Returns true if the file was modified.
func replaceVariablesInFile(filePath string, vars map[string]string) (bool, error) {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", cleanPath, err)
	}

	// Replace each variable
	newData := data
	for key, value := range vars {
		placeholder := "__" + key + "__"
		newData = bytes.ReplaceAll(newData, []byte(placeholder), []byte(value))
	}

	// Only write if changed
	if bytes.Equal(data, newData) {
		return false, nil
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return false, err
	}

	return true, os.WriteFile(cleanPath, newData, info.Mode())
}
