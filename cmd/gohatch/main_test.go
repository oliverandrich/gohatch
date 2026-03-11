// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/oliverandrich/gohatch/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDirectory_NotExists(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	err := validateDirectory(dir)
	assert.NoError(t, err)
}

func TestValidateDirectory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := validateDirectory(dir)
	assert.NoError(t, err)
}

func TestValidateDirectory_NotEmpty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644))

	err := validateDirectory(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func TestValidateDirectory_IsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0o644))

	err := validateDirectory(file)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRunDryRun_GitSource(t *testing.T) {
	oldDir, oldMod, oldExt, oldStrict, oldVars := directory, module, extensions, strict, variables
	defer func() {
		directory, module, extensions, strict, variables = oldDir, oldMod, oldExt, oldStrict, oldVars
	}()

	// Create a local template to simulate fetching
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/user/template\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "cmd", "main.go"), []byte("package main\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, srcDir)
	assert.Contains(t, output, "myapp")
	assert.Contains(t, output, "github.com/me/myapp")
	// Should show module rewrite
	assert.Contains(t, output, "github.com/user/template")
	// Should show file tree
	assert.Contains(t, output, "cmd/main.go")
	assert.Contains(t, output, "go.mod")
}

func TestRunDryRun_LocalSource(t *testing.T) {
	oldDir, oldMod, oldExt, oldStrict, oldVars := directory, module, extensions, strict, variables
	defer func() {
		directory, module, extensions, strict, variables = oldDir, oldMod, oldExt, oldStrict, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/old/tpl\n\ngo 1.21\n"), 0o644))

	directory = "customdir"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, srcDir)
	assert.Contains(t, output, "customdir")
}

func TestRunDryRun_WithExtensions(t *testing.T) {
	oldDir, oldMod, oldExt, oldStrict, oldVars := directory, module, extensions, strict, variables
	defer func() {
		directory, module, extensions, strict, variables = oldDir, oldMod, oldExt, oldStrict, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/old/tpl\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte("key = \"value\"\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = []string{"toml", "yaml"}
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Extensions:")
	assert.Contains(t, output, "toml")
}

func TestParseVariables_DefaultProjectName(t *testing.T) {
	vars := parseVariables(nil, "myapp", "github.com/user/myapp")

	assert.Equal(t, "myapp", vars["ProjectName"])
	assert.Equal(t, "user", vars["GitUser"])
	assert.Len(t, vars, 2)
}

func TestParseVariables_WithVars(t *testing.T) {
	input := []string{"Author=Oliver Andrich", "License=MIT"}
	vars := parseVariables(input, "myapp", "github.com/user/myapp")

	assert.Equal(t, "myapp", vars["ProjectName"])
	assert.Equal(t, "user", vars["GitUser"])
	assert.Equal(t, "Oliver Andrich", vars["Author"])
	assert.Equal(t, "MIT", vars["License"])
	assert.Len(t, vars, 4)
}

func TestParseVariables_OverrideProjectName(t *testing.T) {
	input := []string{"ProjectName=CustomName"}
	vars := parseVariables(input, "myapp", "github.com/user/myapp")

	assert.Equal(t, "CustomName", vars["ProjectName"])
	assert.Equal(t, "user", vars["GitUser"])
	assert.Len(t, vars, 2)
}

func TestParseVariables_ValueWithEquals(t *testing.T) {
	// strings.Cut splits only on the first =, so value keeps the rest
	input := []string{"Equation=a=b+c"}
	vars := parseVariables(input, "myapp", "github.com/user/myapp")

	assert.Equal(t, "a=b+c", vars["Equation"])
}

func TestParseVariables_InvalidEntry(t *testing.T) {
	input := []string{"NoEqualsSign"}
	vars := parseVariables(input, "myapp", "github.com/user/myapp")

	// Should only have default ProjectName and GitUser
	assert.Len(t, vars, 2)
	assert.Equal(t, "myapp", vars["ProjectName"])
	assert.Equal(t, "user", vars["GitUser"])
}

func TestParseVariables_OverrideGitUser(t *testing.T) {
	input := []string{"GitUser=customuser"}
	vars := parseVariables(input, "myapp", "github.com/user/myapp")

	assert.Equal(t, "customuser", vars["GitUser"])
}

func TestParseVariables_ShortModulePath(t *testing.T) {
	// Module path without user (e.g., just "myapp")
	vars := parseVariables(nil, "myapp", "myapp")

	assert.Equal(t, "myapp", vars["ProjectName"])
	_, hasGitUser := vars["GitUser"]
	assert.False(t, hasGitUser, "GitUser should not be set for short module paths")
}

func TestFormatVariables(t *testing.T) {
	vars := map[string]string{
		"Author": "Oliver",
	}
	result := formatVariables(vars)

	assert.Equal(t, "Author=Oliver", result)
}

func TestFormatVariables_Multiple(t *testing.T) {
	vars := map[string]string{
		"A": "1",
		"B": "2",
	}
	result := formatVariables(vars)

	// Order is not guaranteed, but both should be present
	assert.Contains(t, result, "A=1")
	assert.Contains(t, result, "B=2")
	assert.Contains(t, result, ", ")
}

func TestRunDryRun_WithForce(t *testing.T) {
	oldDir, oldMod, oldExt, oldForce, oldVars := directory, module, extensions, force, variables
	defer func() {
		directory, module, extensions, force, variables = oldDir, oldMod, oldExt, oldForce, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# test"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, "--force")
}

func TestRunDryRun_WithNoGitInit(t *testing.T) {
	oldDir, oldMod, oldExt, oldNoGitInit, oldVars := directory, module, extensions, noGitInit, variables
	defer func() {
		directory, module, extensions, noGitInit, variables = oldDir, oldMod, oldExt, oldNoGitInit, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	noGitInit = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.NotContains(t, output, "Would initialize git repository")
}

func TestRunDryRun_DefaultGitInit(t *testing.T) {
	oldDir, oldMod, oldExt, oldNoGitInit, oldVars := directory, module, extensions, noGitInit, variables
	defer func() {
		directory, module, extensions, noGitInit, variables = oldDir, oldMod, oldExt, oldNoGitInit, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	noGitInit = false

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Would initialize git repository with initial commit")
}

func TestVerboseLog_Enabled(t *testing.T) {
	oldVerbose := verbose
	defer func() { verbose = oldVerbose }()

	verbose = true

	output := captureOutput(func() {
		verboseLog("Test message: %s", "value")
	})

	assert.Contains(t, output, "  Test message: value")
}

func TestVerboseLog_Disabled(t *testing.T) {
	oldVerbose := verbose
	defer func() { verbose = oldVerbose }()

	verbose = false

	output := captureOutput(func() {
		verboseLog("Test message: %s", "value")
	})

	assert.Empty(t, output)
}

func TestMergeExtensions_Empty(t *testing.T) {
	result := mergeExtensions(nil, nil)
	assert.Empty(t, result)
}

func TestMergeExtensions_CLIOnly(t *testing.T) {
	result := mergeExtensions([]string{"toml", "yaml"}, nil)
	assert.Equal(t, []string{"toml", "yaml"}, result)
}

func TestMergeExtensions_ConfigOnly(t *testing.T) {
	result := mergeExtensions(nil, []string{"toml", "yaml"})
	assert.Equal(t, []string{"toml", "yaml"}, result)
}

func TestMergeExtensions_Both(t *testing.T) {
	result := mergeExtensions([]string{"md", "txt"}, []string{"toml", "yaml"})
	// Config extensions first, then CLI extensions
	assert.Equal(t, []string{"toml", "yaml", "md", "txt"}, result)
}

func TestMergeExtensions_Deduplication(t *testing.T) {
	result := mergeExtensions([]string{"toml", "yaml"}, []string{"toml", "md"})
	// toml should only appear once (from config)
	assert.Equal(t, []string{"toml", "md", "yaml"}, result)
}

func TestRunDryRun_WithKeepConfig(t *testing.T) {
	oldDir, oldMod, oldExt, oldKeepConfig, oldVars := directory, module, extensions, keepConfig, variables
	defer func() {
		directory, module, extensions, keepConfig, variables = oldDir, oldMod, oldExt, oldKeepConfig, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("extensions = [\"toml\"]\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	keepConfig = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.NotContains(t, output, "Would remove .gohatch.toml")
}

func TestRunDryRun_ConfigRemovalMessage(t *testing.T) {
	oldDir, oldMod, oldExt, oldKeepConfig, oldVars := directory, module, extensions, keepConfig, variables
	defer func() {
		directory, module, extensions, keepConfig, variables = oldDir, oldMod, oldExt, oldKeepConfig, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("extensions = [\"toml\"]\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	keepConfig = false

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Would remove .gohatch.toml")
}

func TestGetGitAuthor(t *testing.T) {
	author := getGitAuthor()

	require.NotNil(t, author)
	assert.NotEmpty(t, author.Name)
	assert.NotEmpty(t, author.Email)
	assert.False(t, author.When.IsZero())
}

func TestInitGitRepo(t *testing.T) {
	dir := t.TempDir()

	// Create a file to commit
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0o644))

	output := captureOutput(func() {
		err := initGitRepo(dir)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Initialized git repository")

	// Verify .git directory exists
	_, err := os.Stat(filepath.Join(dir, ".git"))
	assert.NoError(t, err)
}

func TestValidateGoMod_HasGoMod(t *testing.T) {
	oldDir, oldForce := directory, force
	defer func() { directory, force = oldDir, oldForce }()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0o644))

	directory = dir
	force = false

	err := validateGoMod()
	assert.NoError(t, err)
}

func TestValidateGoMod_NoGoMod_NoForce(t *testing.T) {
	oldDir, oldForce := directory, force
	defer func() { directory, force = oldDir, oldForce }()

	dir := t.TempDir()
	directory = dir
	force = false

	err := validateGoMod()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no go.mod")
}

func TestValidateGoMod_NoGoMod_WithForce(t *testing.T) {
	oldDir, oldForce := directory, force
	defer func() { directory, force = oldDir, oldForce }()

	dir := t.TempDir()
	directory = dir
	force = true

	output := captureOutput(func() {
		err := validateGoMod()
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Warning: template has no go.mod")
}

func TestRenamePaths_EmptyVars(t *testing.T) {
	oldDir := directory
	defer func() { directory = oldDir }()

	directory = t.TempDir()

	err := renamePaths(map[string]string{})
	assert.NoError(t, err)
}

func TestRenamePaths_WithVars(t *testing.T) {
	oldDir, oldVerbose := directory, verbose
	defer func() { directory, verbose = oldDir, oldVerbose }()

	dir := t.TempDir()
	directory = dir
	verbose = false

	// Create a directory with placeholder
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "__ProjectName__"), 0o755))

	output := captureOutput(func() {
		err := renamePaths(map[string]string{"ProjectName": "myapp"})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Renaming paths")

	// Check renamed directory exists
	_, err := os.Stat(filepath.Join(dir, "myapp"))
	assert.NoError(t, err)
}

func TestRewriteModule_NoGoMod(t *testing.T) {
	oldDir := directory
	defer func() { directory = oldDir }()

	directory = t.TempDir()

	err := rewriteModule(nil)
	assert.NoError(t, err)
}

func TestRewriteModule_SameModule(t *testing.T) {
	oldDir, oldMod := directory, module
	defer func() { directory, module = oldDir, oldMod }()

	dir := t.TempDir()
	directory = dir
	module = "github.com/user/test"

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/user/test\n\ngo 1.21\n"), 0o644))

	err := rewriteModule(nil)
	assert.NoError(t, err)
}

func TestRewriteModule_DifferentModule(t *testing.T) {
	oldDir, oldMod, oldVerbose := directory, module, verbose
	defer func() { directory, module, verbose = oldDir, oldMod, oldVerbose }()

	dir := t.TempDir()
	directory = dir
	module = "github.com/me/newapp"
	verbose = false

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/user/template\n\ngo 1.21\n"), 0o644))

	output := captureOutput(func() {
		err := rewriteModule(nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Rewriting module")
	assert.Contains(t, output, "github.com/user/template")
	assert.Contains(t, output, "github.com/me/newapp")
}

func TestReplaceVariables_EmptyVars(t *testing.T) {
	oldDir := directory
	defer func() { directory = oldDir }()

	directory = t.TempDir()

	err := replaceVariables(map[string]string{}, nil)
	assert.NoError(t, err)
}

func TestReplaceVariables_WithVars(t *testing.T) {
	oldDir, oldVerbose := directory, verbose
	defer func() { directory, verbose = oldDir, oldVerbose }()

	dir := t.TempDir()
	directory = dir
	verbose = false

	// Create a Go file with placeholders
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package __ProjectName__\n"), 0o644))

	output := captureOutput(func() {
		err := replaceVariables(map[string]string{"ProjectName": "myapp"}, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Replacing variables")

	// Verify replacement
	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "package myapp")
}

func TestFetchTemplate_LocalSource(t *testing.T) {
	oldDir, oldSrcInput, oldVerbose := directory, srcInput, verbose
	defer func() { directory, srcInput, verbose = oldDir, oldSrcInput, oldVerbose }()

	// Create a source template directory
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".git", "config"), []byte(""), 0o644))

	// Create destination directory
	destDir := filepath.Join(t.TempDir(), "dest")
	directory = destDir
	srcInput = srcDir
	verbose = false

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := fetchTemplate(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Fetching template")

	// Verify go.mod was copied
	_, err := os.Stat(filepath.Join(destDir, "go.mod"))
	require.NoError(t, err)

	// Verify .git was removed
	_, err = os.Stat(filepath.Join(destDir, ".git"))
	assert.True(t, os.IsNotExist(err))
}

func TestExecuteScaffold_FullWorkflow(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\n\nimport \"github.com/template/test/pkg\"\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "pkg", "pkg.go"), []byte("package pkg\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, ".git"), 0o755))

	// Set up globals
	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = false
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Fetching template")
	assert.Contains(t, output, "Rewriting module")
	assert.Contains(t, output, "Created")

	// Verify module was rewritten
	content, err := os.ReadFile(filepath.Join(destDir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "github.com/me/myapp")

	// Verify git repo was initialized
	_, err = os.Stat(filepath.Join(destDir, ".git"))
	assert.NoError(t, err)
}

func TestExecuteScaffold_WithNoGitInit(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	// Verify git repo was NOT initialized
	_, err := os.Stat(filepath.Join(destDir, ".git"))
	assert.True(t, os.IsNotExist(err))
}

func TestExecuteScaffold_DirectoryExists(t *testing.T) {
	oldDir, oldMod, oldSrcInput := directory, module, srcInput
	defer func() { directory, module, srcInput = oldDir, oldMod, oldSrcInput }()

	// Create a non-empty directory
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644))

	directory = dir
	module = "github.com/me/myapp"
	srcInput = "."

	src := &source.LocalSource{Path: "."}

	err := executeScaffold(t.Context(), src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func TestRun_WithDryRun(t *testing.T) {
	oldSrcInput, oldMod, oldDir, oldDryRun, oldVars := srcInput, module, directory, dryRun, variables
	defer func() {
		srcInput, module, directory, dryRun, variables = oldSrcInput, oldMod, oldDir, oldDryRun, oldVars
	}()

	// Create a real template directory
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/tpl/test\n\ngo 1.21\n"), 0o644))

	srcInput = srcDir
	module = "github.com/me/myapp"
	directory = ""
	dryRun = true
	variables = nil

	output := captureOutput(func() {
		err := run(t.Context(), nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
}

func TestExecuteScaffold_WithConfigFile(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with config
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("extensions = [\"toml\"]\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = true
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Found .gohatch.toml")
	assert.Contains(t, output, "Removed .gohatch.toml")

	// Verify config was removed
	_, err := os.Stat(filepath.Join(destDir, ".gohatch.toml"))
	assert.True(t, os.IsNotExist(err))
}

func TestExecuteScaffold_KeepConfig(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with config
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("extensions = [\"toml\"]\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = true
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	// Verify config was kept
	_, err := os.Stat(filepath.Join(destDir, ".gohatch.toml"))
	assert.NoError(t, err)
}

func TestExecuteScaffold_UnsetVarInPath(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with unset variable in path
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "__Author__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "__Author__", "main.go"), []byte("package main\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unset template variables in paths")
		assert.Contains(t, err.Error(), "Author")
	})

	// Directory should be cleaned up
	_, err := os.Stat(destDir)
	assert.True(t, os.IsNotExist(err), "directory should be removed on error")
}

func TestExecuteScaffold_UnsetVarInContent_NoStrict(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with unset variable in content
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nconst Author = \"__Author__\"\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Warning: unset template variable __Author__")

	// Variable should be removed (replaced with empty string)
	content, err := os.ReadFile(filepath.Join(destDir, "main.go"))
	require.NoError(t, err)
	assert.NotContains(t, string(content), "__Author__")
	assert.Contains(t, string(content), `const Author = ""`)
}

func TestExecuteScaffold_UnsetVarInContent_Strict(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with unset variable in content
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nconst Author = \"__Author__\"\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = true
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unset template variables in file contents")
		assert.Contains(t, err.Error(), "--strict mode")
		assert.Contains(t, err.Error(), "Author")
	})

	// Directory should be cleaned up
	_, err := os.Stat(destDir)
	assert.True(t, os.IsNotExist(err), "directory should be removed on error")
}

func TestExecuteScaffold_AllVarsSet(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template where all variables are provided
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nconst Author = \"__Author__\"\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = []string{"Author=Oliver"}
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = true
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	// No warnings should be shown
	assert.NotContains(t, output, "Warning: unset template variable")

	// Variable should be replaced
	content, err := os.ReadFile(filepath.Join(destDir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `const Author = "Oliver"`)
}

func TestRunDryRun_WithStrict(t *testing.T) {
	oldDir, oldMod, oldExt, oldStrict, oldVars := directory, module, extensions, strict, variables
	defer func() {
		directory, module, extensions, strict, variables = oldDir, oldMod, oldExt, oldStrict, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	strict = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "--strict")
}

// =============================================================================
// Error path tests for run, initGitRepo, getGitAuthor, fetchTemplate,
// executeScaffold
// =============================================================================

func TestRun_ParseSourceError(t *testing.T) {
	oldSrcInput, oldMod, oldDir, oldDryRun := srcInput, module, directory, dryRun
	defer func() { srcInput, module, directory, dryRun = oldSrcInput, oldMod, oldDir, oldDryRun }()

	// A local path with version suffix triggers a source.Parse error
	srcInput = "./template@v1.0.0"
	module = "github.com/me/myapp"
	directory = ""
	dryRun = false

	err := run(t.Context(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing source")
}

func TestGetGitAuthor_Fallback(t *testing.T) {
	// Set HOME to an empty temp dir where no .gitconfig exists
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	author := getGitAuthor()

	require.NotNil(t, author)
	assert.Equal(t, "gohatch", author.Name)
	assert.Equal(t, "gohatch@localhost", author.Email)
	assert.False(t, author.When.IsZero())
}

func TestInitGitRepo_InvalidPath(t *testing.T) {
	// Use a file path instead of a directory — PlainInitWithOptions should fail
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notadir.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644))

	captureOutput(func() {
		err := initGitRepo(filePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git init")
	})
}

func TestFetchTemplate_FetchError(t *testing.T) {
	oldDir, oldSrcInput := directory, srcInput
	defer func() { directory, srcInput = oldDir, oldSrcInput }()

	directory = filepath.Join(t.TempDir(), "dest")
	srcInput = "invalid-source"

	// Use a GitSource with an invalid URL so Fetch fails
	src := &source.GitSource{URL: "file:///nonexistent/repo"}

	captureOutput(func() {
		err := fetchTemplate(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching template")
	})
}

func TestExecuteScaffold_FetchError(t *testing.T) {
	oldDir, oldMod, oldSrcInput, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldStrict :=
		directory, module, srcInput, variables, force, noGitInit, keepConfig, verbose, strict
	defer func() {
		directory, module, srcInput, variables, force, noGitInit, keepConfig, verbose, strict =
			oldDir, oldMod, oldSrcInput, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldStrict
	}()

	directory = filepath.Join(t.TempDir(), "dest")
	module = "github.com/me/myapp"
	srcInput = "invalid"
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false

	src := &source.GitSource{URL: "file:///nonexistent/repo"}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching template")
	})
}

func TestExecuteScaffold_NoGoMod_NoForce(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Template without go.mod and no --force
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# Template"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no go.mod")
	})
}

func TestExecuteScaffold_NoGoMod_WithForce(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Template without go.mod but with --force
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# Template"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = true
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Warning: template has no go.mod")
	assert.Contains(t, output, "Created")
}

func TestExecuteScaffold_WithVariablesAndPaths(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Template with variables in paths and content
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "cmd", "__ProjectName__"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "cmd", "__ProjectName__", "main.go"),
		[]byte("package main\n\nconst app = \"__ProjectName__\"\n"),
		0o644,
	))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Renaming paths")
	assert.Contains(t, output, "Replacing variables")

	// Verify path was renamed
	assert.DirExists(t, filepath.Join(destDir, "cmd", "myapp"))

	// Verify variable was replaced
	content, err := os.ReadFile(filepath.Join(destDir, "cmd", "myapp", "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(content), `"myapp"`)
}

func TestExecuteScaffold_MalformedConfig(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict
	}()

	// Create a source template with a malformed .gohatch.toml
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("this is not valid toml {{{"), 0o644))

	destDir := filepath.Join(t.TempDir(), "myapp")
	directory = destDir
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = false
	noGitInit = true
	keepConfig = false
	verbose = false
	strict = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading config")
	})
}

func TestRunDryRun_PathRenamePreview(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars := directory, module, extensions, variables
	defer func() {
		directory, module, extensions, variables = oldDir, oldMod, oldExt, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/tpl/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "cmd", "__ProjectName__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "cmd", "__ProjectName__", "main.go"), []byte("package main\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Renamed paths:")
	assert.Contains(t, output, "__ProjectName__")
	assert.Contains(t, output, "myapp")
}

func TestRunDryRun_UnsetVarWarning(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars := directory, module, extensions, variables
	defer func() {
		directory, module, extensions, variables = oldDir, oldMod, oldExt, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/tpl/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nconst author = \"__Author__\"\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Warning")
	assert.Contains(t, output, "__Author__")
}

func TestRunDryRun_ModuleRewriteDisplay(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars := directory, module, extensions, variables
	defer func() {
		directory, module, extensions, variables = oldDir, oldMod, oldExt, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/old/template\n\ngo 1.21\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Module:")
	assert.Contains(t, output, "github.com/old/template")
	assert.Contains(t, output, "github.com/me/myapp")
}

func TestRunDryRun_NoGoMod(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce := directory, module, extensions, variables, force
	defer func() {
		directory, module, extensions, variables, force = oldDir, oldMod, oldExt, oldVars, oldForce
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# test"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	force = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "no go.mod")
	assert.NotContains(t, output, "Module:")
}

func TestRunDryRun_WithConfigExtensions(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars := directory, module, extensions, variables
	defer func() {
		directory, module, extensions, variables = oldDir, oldMod, oldExt, oldVars
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("extensions = [\"toml\", \"justfile\"]\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Extensions:")
	assert.Contains(t, output, "toml")
	assert.Contains(t, output, "justfile")
}

func TestRunDryRun_FetchError(t *testing.T) {
	oldDir, oldMod, oldVars := directory, module, variables
	defer func() {
		directory, module, variables = oldDir, oldMod, oldVars
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	variables = nil

	src := &source.GitSource{URL: "file:///nonexistent/repo"}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetching template")
	})

	_ = output
}
