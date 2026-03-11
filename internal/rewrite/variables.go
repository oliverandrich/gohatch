// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Variables replaces template variables in all files.
// Variables use dunder-style syntax: __VariableName__ → value.
// Returns the list of modified files.
func Variables(dir string, vars map[string]string, extraPatterns []string) ([]string, error) {
	if len(vars) == 0 {
		return nil, nil
	}

	var modifiedFiles []string

	// Build pattern set: go + extra patterns
	patternSet := parseFilePatterns(extraPatterns)
	patternSet["go"] = true

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("opening directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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

		// Check if file matches by extension or name
		if !matchesFilePattern(d.Name(), patternSet) {
			return nil
		}

		relPath, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		modified, replaceErr := replaceVariablesInFile(root, relPath, vars)
		if replaceErr != nil {
			return replaceErr
		}
		if modified {
			modifiedFiles = append(modifiedFiles, relPath)
		}
		return nil
	})

	return modifiedFiles, err
}

// replaceVariablesInFile replaces __Key__ with Value for all variables.
// Returns true if the file was modified.
func replaceVariablesInFile(root *os.Root, relPath string, vars map[string]string) (bool, error) {
	data, err := readFromRoot(root, relPath)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", relPath, err)
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

	info, err := root.Stat(relPath)
	if err != nil {
		return false, err
	}

	return true, writeToRoot(root, relPath, newData, info.Mode())
}
