// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"codeberg.org/oliverandrich/gohatch/internal/source"
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
	// Save and restore global state
	oldDir, oldMod, oldExt := directory, module, extensions
	defer func() {
		directory, module, extensions = oldDir, oldMod, oldExt
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil

	src := &source.GitSource{
		URL:     "https://github.com/user/template",
		Version: "v1.0.0",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, "https://github.com/user/template")
	assert.Contains(t, output, "v1.0.0")
	assert.Contains(t, output, "myapp")
	assert.Contains(t, output, "github.com/me/myapp")
}

func TestRunDryRun_LocalSource(t *testing.T) {
	oldDir, oldMod, oldExt := directory, module, extensions
	defer func() {
		directory, module, extensions = oldDir, oldMod, oldExt
	}()

	directory = "customdir"
	module = "github.com/me/myapp"
	extensions = nil

	src := &source.LocalSource{
		Path: "./my-template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, "./my-template (local)")
	assert.Contains(t, output, "customdir")
}

func TestRunDryRun_WithExtensions(t *testing.T) {
	oldDir, oldMod, oldExt := directory, module, extensions
	defer func() {
		directory, module, extensions = oldDir, oldMod, oldExt
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = []string{"toml", "yaml"}

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "CLI Extensions: [toml yaml]")
	assert.Contains(t, output, "files with specified extensions")
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
	oldDir, oldMod, oldExt, oldForce := directory, module, extensions, force
	defer func() {
		directory, module, extensions, force = oldDir, oldMod, oldExt, oldForce
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	force = true

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
	assert.Contains(t, output, "--force")
}

func TestRunDryRun_WithNoGitInit(t *testing.T) {
	oldDir, oldMod, oldExt, oldNoGitInit := directory, module, extensions, noGitInit
	defer func() {
		directory, module, extensions, noGitInit = oldDir, oldMod, oldExt, oldNoGitInit
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	noGitInit = true

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "--no-git-init")
	assert.NotContains(t, output, "Would initialize git repository")
}

func TestRunDryRun_DefaultGitInit(t *testing.T) {
	oldDir, oldMod, oldExt, oldNoGitInit := directory, module, extensions, noGitInit
	defer func() {
		directory, module, extensions, noGitInit = oldDir, oldMod, oldExt, oldNoGitInit
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	noGitInit = false

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
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
	oldDir, oldMod, oldExt, oldKeepConfig := directory, module, extensions, keepConfig
	defer func() {
		directory, module, extensions, keepConfig = oldDir, oldMod, oldExt, oldKeepConfig
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	keepConfig = true

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "--keep-config")
	assert.NotContains(t, output, "Would remove .gohatch.toml")
}

func TestRunDryRun_ConfigRemovalMessage(t *testing.T) {
	oldDir, oldMod, oldExt, oldKeepConfig := directory, module, extensions, keepConfig
	defer func() {
		directory, module, extensions, keepConfig = oldDir, oldMod, oldExt, oldKeepConfig
	}()

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	keepConfig = false

	src := &source.GitSource{
		URL: "https://github.com/user/template",
	}

	output := captureOutput(func() {
		err := runDryRun(src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Would remove .gohatch.toml")
	assert.Contains(t, output, "Would read .gohatch.toml")
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput
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
	oldSrcInput, oldMod, oldDir, oldDryRun := srcInput, module, directory, dryRun
	defer func() { srcInput, module, directory, dryRun = oldSrcInput, oldMod, oldDir, oldDryRun }()

	srcInput = "./template"
	module = "github.com/me/myapp"
	directory = ""
	dryRun = true

	output := captureOutput(func() {
		err := run(t.Context(), nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run mode")
}

func TestExecuteScaffold_WithConfigFile(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput
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
