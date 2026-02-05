// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectUnsetVars_NoUnsetVars(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), `package main
const Name = "__ProjectName__"
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InPaths) != 0 {
		t.Errorf("InPaths = %v, want empty", result.InPaths)
	}
	if len(result.InContents) != 0 {
		t.Errorf("InContents = %v, want empty", result.InContents)
	}
}

func TestDetectUnsetVars_UnsetInPath(t *testing.T) {
	tmpDir := t.TempDir()

	varDir := filepath.Join(tmpDir, "__Author__")
	if err := os.MkdirAll(varDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(varDir, "main.go"), "package main\n")

	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InPaths) != 1 || result.InPaths[0] != "Author" {
		t.Errorf("InPaths = %v, want [Author]", result.InPaths)
	}
}

func TestDetectUnsetVars_UnsetInContent(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), `package main
const Author = "__Author__"
const License = "__License__"
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 2 {
		t.Fatalf("InContents = %v, want [Author License]", result.InContents)
	}
	if result.InContents[0] != "Author" || result.InContents[1] != "License" {
		t.Errorf("InContents = %v, want [Author License]", result.InContents)
	}
}

func TestDetectUnsetVars_SetVarsNotReported(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), `package main
const Name = "__ProjectName__"
const Author = "__Author__"
`)
	vars := map[string]string{"ProjectName": "myapp", "Author": "Oliver"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InPaths) != 0 {
		t.Errorf("InPaths = %v, want empty", result.InPaths)
	}
	if len(result.InContents) != 0 {
		t.Errorf("InContents = %v, want empty", result.InContents)
	}
}

func TestDetectUnsetVars_ExtraPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "config.toml"), `name = "__Author__"
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, []string{"toml"})
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 1 || result.InContents[0] != "Author" {
		t.Errorf("InContents = %v, want [Author]", result.InContents)
	}
}

func TestDetectUnsetVars_NonMatchingFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	// .txt file without extra patterns should be ignored
	writeFile(t, filepath.Join(tmpDir, "readme.txt"), `Hello __Author__
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 0 {
		t.Errorf("InContents = %v, want empty", result.InContents)
	}
}

func TestDetectUnsetVars_SkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	vendorDir := filepath.Join(tmpDir, "vendor", "pkg")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(vendorDir, "pkg.go"), `package pkg
const Name = "__Author__"
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 0 {
		t.Errorf("InContents = %v, want empty (vendor should be skipped)", result.InContents)
	}
}

func TestDetectUnsetVars_SkipsGitDir(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(gitDir, "config"), `__Author__
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 0 {
		t.Errorf("InContents = %v, want empty (.git should be skipped)", result.InContents)
	}
	if len(result.InPaths) != 0 {
		t.Errorf("InPaths = %v, want empty (.git should be skipped)", result.InPaths)
	}
}

func TestDetectUnsetVars_SortedOutput(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), `package main
const Z = "__Zebra__"
const A = "__Alpha__"
const M = "__Middle__"
`)
	vars := map[string]string{}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 3 {
		t.Fatalf("InContents = %v, want 3 items", result.InContents)
	}
	if result.InContents[0] != "Alpha" || result.InContents[1] != "Middle" || result.InContents[2] != "Zebra" {
		t.Errorf("InContents = %v, want [Alpha Middle Zebra]", result.InContents)
	}
}

func TestDetectUnsetVars_Deduplicated(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), `package main
const A = "__Author__"
const B = "__Author__"
`)
	vars := map[string]string{}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InContents) != 1 || result.InContents[0] != "Author" {
		t.Errorf("InContents = %v, want [Author]", result.InContents)
	}
}

func TestDetectUnsetVars_BothPathAndContent(t *testing.T) {
	tmpDir := t.TempDir()

	varDir := filepath.Join(tmpDir, "__Author__")
	if err := os.MkdirAll(varDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(varDir, "main.go"), `package main
const License = "__License__"
`)
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InPaths) != 1 || result.InPaths[0] != "Author" {
		t.Errorf("InPaths = %v, want [Author]", result.InPaths)
	}
	if len(result.InContents) != 1 || result.InContents[0] != "License" {
		t.Errorf("InContents = %v, want [License]", result.InContents)
	}
}

func TestVarPattern_ValidPatterns(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"__Author__", "Author"},
		{"__ProjectName__", "ProjectName"},
		{"__A__", "A"},
		{"__Var_Name__", "Var_Name"},
		{"__X1__", "X1"},
	}

	for _, tt := range tests {
		matches := varPattern.FindStringSubmatch(tt.input)
		if len(matches) < 2 || matches[1] != tt.want {
			t.Errorf("varPattern.FindStringSubmatch(%q) = %v, want [_, %q]", tt.input, matches, tt.want)
		}
	}
}

func TestVarPattern_InvalidPatterns(t *testing.T) {
	tests := []string{
		"__1invalid__", // starts with digit
		"____",         // empty name
		"_single_",     // single underscores
		"__",           // just underscores
		"no vars here", // no pattern
	}

	for _, input := range tests {
		matches := varPattern.FindStringSubmatch(input)
		if len(matches) > 0 {
			t.Errorf("varPattern should not match %q, but got %v", input, matches)
		}
	}
}

func TestDetectUnsetVars_FileInPathWithVar(t *testing.T) {
	tmpDir := t.TempDir()

	// File name (not directory) with variable
	writeFile(t, filepath.Join(tmpDir, "__Author__.go"), "package main\n")
	vars := map[string]string{"ProjectName": "myapp"}

	result, err := DetectUnsetVars(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("DetectUnsetVars() error = %v", err)
	}
	if len(result.InPaths) != 1 || result.InPaths[0] != "Author" {
		t.Errorf("InPaths = %v, want [Author]", result.InPaths)
	}
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
