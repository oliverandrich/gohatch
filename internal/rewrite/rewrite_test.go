// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModule(t *testing.T) {
	// Create a temporary directory with a mock Go project
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/old/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a .go file with imports
	goFile := `package main

import (
	"fmt"

	"github.com/old/module/internal/foo"
	"github.com/old/module/pkg/bar"
)

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run Module rewrite
	_, err := Module(tmpDir, "github.com/new/project", nil)
	if err != nil {
		t.Fatalf("Module() error = %v", err)
	}

	// Verify go.mod was updated
	data, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "module github.com/new/project") {
		t.Errorf("go.mod not updated, got: %s", string(data))
	}

	// Verify .go file imports were updated
	data, err = os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `"github.com/new/project/internal/foo"`) {
		t.Errorf("import not updated, got: %s", content)
	}
	if !strings.Contains(content, `"github.com/new/project/pkg/bar"`) {
		t.Errorf("import not updated, got: %s", content)
	}
}

func TestModuleSameModule(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/same/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should return nil without changes when module is the same
	_, err := Module(tmpDir, "github.com/same/module", nil)
	if err != nil {
		t.Fatalf("Module() error = %v", err)
	}
}

func TestModuleWithExtraExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module github.com/old/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a .toml file with module reference
	tomlFile := `[package]
name = "myapp"
repository = "github.com/old/module"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tomlFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a .yaml file with module reference
	yamlFile := `module: github.com/old/module
version: 1.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(yamlFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run Module rewrite with extra extensions
	_, err := Module(tmpDir, "github.com/new/project", []string{"toml", "yaml"})
	if err != nil {
		t.Fatalf("Module() error = %v", err)
	}

	// Verify .toml file was updated
	data, err := os.ReadFile(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "github.com/new/project") {
		t.Errorf("toml not updated, got: %s", string(data))
	}

	// Verify .yaml file was updated
	data, err = os.ReadFile(filepath.Join(tmpDir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "github.com/new/project") {
		t.Errorf("yaml not updated, got: %s", string(data))
	}
}

func TestModuleWithExtensionDotPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/old/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	shFile := `#!/bin/bash
# github.com/old/module
echo "hello"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "script.sh"), []byte(shFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Extensions with dot prefix should also work
	_, err := Module(tmpDir, "github.com/new/project", []string{".sh"})
	if err != nil {
		t.Fatalf("Module() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "script.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "github.com/new/project") {
		t.Errorf("sh not updated, got: %s", string(data))
	}
}

func TestModuleSkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/old/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create vendor directory with a .go file
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "other", "pkg")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}

	vendorFile := `package pkg

import "github.com/old/module/internal"
`
	if err := os.WriteFile(filepath.Join(vendorDir, "pkg.go"), []byte(vendorFile), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Module(tmpDir, "github.com/new/project", nil)
	if err != nil {
		t.Fatalf("Module() error = %v", err)
	}

	// Vendor file should NOT be modified
	data, err := os.ReadFile(filepath.Join(vendorDir, "pkg.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "github.com/new/project") {
		t.Errorf("vendor file should not be modified, got: %s", string(data))
	}
}

func TestReadModulePath(t *testing.T) {
	tmpDir := t.TempDir()

	goMod := `module github.com/test/module

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := ReadModulePath(tmpDir)
	if err != nil {
		t.Fatalf("ReadModulePath() error = %v", err)
	}
	if path != "github.com/test/module" {
		t.Errorf("ReadModulePath() = %q, want %q", path, "github.com/test/module")
	}
}

func TestReadModulePathNoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ReadModulePath(tmpDir)
	if err == nil {
		t.Error("ReadModulePath() should error when go.mod doesn't exist")
	}
}

func TestHasGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Without go.mod
	if HasGoMod(tmpDir) {
		t.Error("HasGoMod() = true, want false")
	}

	// With go.mod
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasGoMod(tmpDir) {
		t.Error("HasGoMod() = false, want true")
	}
}

func TestRewriteFileNoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// File with no matching imports
	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Get original mod time
	origInfo, _ := os.Stat(filePath)

	_, err := rewriteGoFile(filePath, "github.com/other/module", "github.com/new/module")
	if err != nil {
		t.Fatalf("rewriteGoFile() error = %v", err)
	}

	// File should not be modified (check content is same)
	data, _ := os.ReadFile(filePath)
	if string(data) != goFile {
		t.Errorf("file was modified when it shouldn't be")
	}

	// Mod time should be unchanged (file wasn't written)
	newInfo, _ := os.Stat(filePath)
	if !origInfo.ModTime().Equal(newInfo.ModTime()) {
		t.Errorf("file was rewritten when it shouldn't be")
	}
}

func TestRewriteTextFileNoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	content := `some content without module reference`
	filePath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := rewriteTextFile(filePath, "github.com/old/module", "github.com/new/module")
	if err != nil {
		t.Fatalf("rewriteTextFile() error = %v", err)
	}

	data, _ := os.ReadFile(filePath)
	if string(data) != content {
		t.Errorf("file was modified when it shouldn't be")
	}
}

func TestVariables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .go file with variables
	goFile := `package main

const ProjectName = "__ProjectName__"
const Author = "__Author__"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goFile), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a .toml file with variables
	tomlFile := `[project]
name = "__ProjectName__"
author = "__Author__"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tomlFile), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"ProjectName": "MyApp",
		"Author":      "Oliver Andrich",
	}

	_, err := Variables(tmpDir, vars, []string{"toml"})
	if err != nil {
		t.Fatalf("Variables() error = %v", err)
	}

	// Check .go file
	data, err := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `const ProjectName = "MyApp"`) {
		t.Errorf(".go file: ProjectName not replaced, got: %s", data)
	}
	if !strings.Contains(string(data), `const Author = "Oliver Andrich"`) {
		t.Errorf(".go file: Author not replaced, got: %s", data)
	}

	// Check .toml file
	data, err = os.ReadFile(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `name = "MyApp"`) {
		t.Errorf(".toml file: ProjectName not replaced, got: %s", data)
	}
	if !strings.Contains(string(data), `author = "Oliver Andrich"`) {
		t.Errorf(".toml file: Author not replaced, got: %s", data)
	}
}

func TestVariablesEmptyMap(t *testing.T) {
	tmpDir := t.TempDir()

	// Should return nil immediately for empty map
	_, err := Variables(tmpDir, map[string]string{}, nil)
	if err != nil {
		t.Fatalf("Variables() error = %v", err)
	}
}

func TestVariablesNoChanges(t *testing.T) {
	tmpDir := t.TempDir()

	content := `package main

const Name = "test"
`
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"ProjectName": "MyApp",
	}

	_, err := Variables(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("Variables() error = %v", err)
	}

	// File should not be modified (no matching placeholders)
	data, _ := os.ReadFile(filePath)
	if string(data) != content {
		t.Errorf("file was modified when it shouldn't be")
	}
}

func TestVariablesSkipsVendor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create vendor directory with a file containing variables
	vendorDir := filepath.Join(tmpDir, "vendor", "pkg")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}

	vendorFile := `package pkg
const Name = "__ProjectName__"
`
	if err := os.WriteFile(filepath.Join(vendorDir, "pkg.go"), []byte(vendorFile), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"ProjectName": "MyApp",
	}

	_, err := Variables(tmpDir, vars, nil)
	if err != nil {
		t.Fatalf("Variables() error = %v", err)
	}

	// Vendor file should NOT be modified
	data, err := os.ReadFile(filepath.Join(vendorDir, "pkg.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "MyApp") {
		t.Errorf("vendor file should not be modified, got: %s", data)
	}
}

func TestRenamePaths_SimpleDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with variable in name
	varDir := filepath.Join(tmpDir, "cmd", "__ProjectName__")
	if err := os.MkdirAll(varDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file inside
	if err := os.WriteFile(filepath.Join(varDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{"ProjectName": "myapp"}
	renamed, err := RenamePaths(tmpDir, vars)
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	// Should have renamed the directory
	if len(renamed) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(renamed), renamed)
	}

	// New path should exist
	newDir := filepath.Join(tmpDir, "cmd", "myapp")
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Errorf("renamed directory should exist at %s", newDir)
	}

	// Old path should not exist
	if _, err := os.Stat(varDir); !os.IsNotExist(err) {
		t.Errorf("old directory should not exist at %s", varDir)
	}

	// File inside should still exist
	if _, err := os.Stat(filepath.Join(newDir, "main.go")); os.IsNotExist(err) {
		t.Errorf("file inside renamed directory should exist")
	}
}

func TestRenamePaths_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directories with variables
	nestedDir := filepath.Join(tmpDir, "__A__", "__B__")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file inside
	if err := os.WriteFile(filepath.Join(nestedDir, "file.go"), []byte("package pkg"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{"A": "first", "B": "second"}
	renamed, err := RenamePaths(tmpDir, vars)
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	// Should have renamed both directories
	if len(renamed) != 2 {
		t.Errorf("expected 2 renames, got %d: %v", len(renamed), renamed)
	}

	// New path should exist
	newPath := filepath.Join(tmpDir, "first", "second", "file.go")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("file should exist at %s", newPath)
	}
}

func TestRenamePaths_FileRename(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with variable in name
	if err := os.WriteFile(filepath.Join(tmpDir, "__ProjectName__.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{"ProjectName": "myapp"}
	renamed, err := RenamePaths(tmpDir, vars)
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	if len(renamed) != 1 {
		t.Errorf("expected 1 rename, got %d: %v", len(renamed), renamed)
	}

	// New file should exist
	if _, err := os.Stat(filepath.Join(tmpDir, "myapp.go")); os.IsNotExist(err) {
		t.Errorf("renamed file should exist")
	}

	// Old file should not exist
	if _, err := os.Stat(filepath.Join(tmpDir, "__ProjectName__.go")); !os.IsNotExist(err) {
		t.Errorf("old file should not exist")
	}
}

func TestRenamePaths_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a normal directory and file
	if err := os.MkdirAll(filepath.Join(tmpDir, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cmd", "app", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{"ProjectName": "myapp"}
	renamed, err := RenamePaths(tmpDir, vars)
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	// Should have no renames
	if len(renamed) != 0 {
		t.Errorf("expected 0 renames, got %d: %v", len(renamed), renamed)
	}
}

func TestRenamePaths_EmptyVars(t *testing.T) {
	tmpDir := t.TempDir()

	renamed, err := RenamePaths(tmpDir, map[string]string{})
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	if renamed != nil {
		t.Errorf("expected nil, got %v", renamed)
	}
}

func TestRenamePaths_DirectoryAndFileWithSameVar(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with variable
	varDir := filepath.Join(tmpDir, "__ProjectName__")
	if err := os.MkdirAll(varDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create file with variable inside
	if err := os.WriteFile(filepath.Join(varDir, "__ProjectName__.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{"ProjectName": "myapp"}
	renamed, err := RenamePaths(tmpDir, vars)
	if err != nil {
		t.Fatalf("RenamePaths() error = %v", err)
	}

	// Should have renamed both
	if len(renamed) != 2 {
		t.Errorf("expected 2 renames, got %d: %v", len(renamed), renamed)
	}

	// Final file should exist
	finalPath := filepath.Join(tmpDir, "myapp", "myapp.go")
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		t.Errorf("final file should exist at %s", finalPath)
	}
}
