// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"path/filepath"
	"strings"
)

// parseFilePatterns normalizes patterns by removing leading dots.
// Each pattern is treated as both a potential filename and extension.
func parseFilePatterns(patterns []string) map[string]bool {
	result := make(map[string]bool)
	for _, p := range patterns {
		p = strings.TrimPrefix(p, ".")
		if p != "" {
			result[p] = true
		}
	}
	return result
}

// matchesFilePattern checks if a filename matches any pattern.
// A pattern matches if the filename equals it exactly, or if the
// file's extension (without leading dot) equals the pattern.
func matchesFilePattern(name string, patterns map[string]bool) bool {
	// Check exact filename match
	if patterns[name] {
		return true
	}
	// Check extension match (without leading dot)
	ext := filepath.Ext(name)
	if ext != "" {
		return patterns[strings.TrimPrefix(ext, ".")]
	}
	return false
}
