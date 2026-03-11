// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// varPattern matches dunder-style template variables like __VarName__.
var varPattern = regexp.MustCompile(`__([A-Za-z][A-Za-z0-9_]*)__`)

// UnsetVars holds the names of unset template variables found in paths and file contents.
type UnsetVars struct {
	InPaths    []string
	InContents []string
}

// DetectUnsetVars scans the directory for template variables that are not present
// in the vars map. It returns variable names found in path names and file contents
// separately, sorted and deduplicated.
func DetectUnsetVars(dir string, vars map[string]string, extraPatterns []string) (*UnsetVars, error) {
	pathVars := make(map[string]bool)
	contentVars := make(map[string]bool)

	// Build pattern set: go + extra patterns (same as Variables())
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

		// Skip special directories
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return filepath.SkipDir
			}
		}

		// Check path name for unset variables
		matches := varPattern.FindAllStringSubmatch(d.Name(), -1)
		for _, m := range matches {
			name := m[1]
			if _, set := vars[name]; !set {
				pathVars[name] = true
			}
		}

		// Check file contents for unset variables
		if !d.IsDir() && matchesFilePattern(d.Name(), patternSet) {
			relPath, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return relErr
			}
			data, readErr := readFromRoot(root, relPath)
			if readErr != nil {
				return readErr
			}
			contentMatches := varPattern.FindAllStringSubmatch(string(data), -1)
			for _, m := range contentMatches {
				name := m[1]
				if _, set := vars[name]; !set {
					contentVars[name] = true
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &UnsetVars{
		InPaths:    sortedKeys(pathVars),
		InContents: sortedKeys(contentVars),
	}, nil
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
