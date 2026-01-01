// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RenamePaths renames directories and files containing template variables.
// Variables use dunder-style syntax: __VariableName__ in path names.
// Returns the list of renamed paths (formatted as "old → new").
func RenamePaths(dir string, vars map[string]string) ([]string, error) {
	if len(vars) == 0 {
		return nil, nil
	}

	// Phase 1: Collect all paths that need renaming
	renames, err := collectPathsToRename(dir, vars)
	if err != nil {
		return nil, fmt.Errorf("collecting paths: %w", err)
	}

	if len(renames) == 0 {
		return nil, nil
	}

	// Phase 2: Sort by depth (deepest first) and rename
	paths := make([]string, 0, len(renames))
	for p := range renames {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool {
		return strings.Count(paths[i], string(os.PathSeparator)) >
			strings.Count(paths[j], string(os.PathSeparator))
	})

	renamedPaths := make([]string, 0, len(paths))
	for _, oldPath := range paths {
		newPath := renames[oldPath]

		// The parent directory might have been renamed already,
		// so we need to update the paths accordingly
		oldPath = updatePathWithRenames(oldPath, renamedPaths, dir)
		newPath = updatePathWithRenames(newPath, renamedPaths, dir)

		if err := os.Rename(oldPath, newPath); err != nil {
			return nil, fmt.Errorf("renaming %s to %s: %w", oldPath, newPath, err)
		}

		oldRel, _ := filepath.Rel(dir, oldPath)
		newRel, _ := filepath.Rel(dir, newPath)
		renamedPaths = append(renamedPaths, oldRel+" → "+newRel)
	}

	return renamedPaths, nil
}

// collectPathsToRename walks the directory tree and collects paths that contain
// template variables in their names.
func collectPathsToRename(dir string, vars map[string]string) (map[string]string, error) {
	renames := make(map[string]string)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip special directories
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == ".git") {
			return filepath.SkipDir
		}

		name := d.Name()
		newName := name
		for key, value := range vars {
			placeholder := "__" + key + "__"
			newName = strings.ReplaceAll(newName, placeholder, value)
		}

		if newName != name {
			newPath := filepath.Join(filepath.Dir(path), newName)
			renames[path] = newPath
		}
		return nil
	})

	return renames, err
}

// updatePathWithRenames updates a path based on previously completed renames.
// This handles the case where a parent directory was renamed before its children.
func updatePathWithRenames(path string, renamedPaths []string, baseDir string) string {
	for _, rename := range renamedPaths {
		parts := strings.Split(rename, " → ")
		if len(parts) != 2 {
			continue
		}
		oldRel, newRel := parts[0], parts[1]
		oldAbs := filepath.Join(baseDir, oldRel)
		newAbs := filepath.Join(baseDir, newRel)

		// If the path starts with the old path, replace it
		if strings.HasPrefix(path, oldAbs+string(os.PathSeparator)) {
			path = newAbs + strings.TrimPrefix(path, oldAbs)
		} else if path == oldAbs {
			path = newAbs
		}
	}
	return path
}
