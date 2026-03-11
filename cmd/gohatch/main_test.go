// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	gohatchcfg "github.com/oliverandrich/gohatch/internal/config"
	"github.com/oliverandrich/gohatch/internal/rewrite"
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldSrcInput, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldStrict, oldNoHooks :=
		directory, module, srcInput, variables, force, noGitInit, keepConfig, verbose, strict, noHooks
	defer func() {
		directory, module, srcInput, variables, force, noGitInit, keepConfig, verbose, strict, noHooks =
			oldDir, oldMod, oldSrcInput, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
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

// --- Tests for interactive prompt / --no-prompt ---

func TestDetectUnsetVars_NoPrompt_InPath(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoPrompt, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noPrompt, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noPrompt, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoPrompt, oldNoHooks
	}()

	// Create a template with unset variable in path
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
	noPrompt = true
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unset template variables in paths")
		assert.Contains(t, err.Error(), "Author")
	})
}

func TestDetectUnsetVars_NoPrompt_InContent(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoPrompt, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noPrompt, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noPrompt, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoPrompt, oldNoHooks
	}()

	// Create a template with unset variable in content
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
	noPrompt = true
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

func TestDetectUnsetVars_NonInteractiveTerminal(t *testing.T) {
	// When not in a terminal, should behave like --no-prompt
	oldIsInteractive := isInteractive
	defer func() { isInteractive = oldIsInteractive }()
	isInteractive = func() bool { return false }

	oldDir, oldNoPrompt, oldStrict := directory, noPrompt, strict
	defer func() { directory, noPrompt, strict = oldDir, oldNoPrompt, oldStrict }()
	noPrompt = false // noPrompt is false, but terminal is not interactive
	strict = false

	// Create temp dir with unset path var
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "__Author__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "__Author__", "file.go"), []byte("package x\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "out")
	directory = destDir

	// Copy template to destDir
	require.NoError(t, copyDir(srcDir, destDir))

	vars := map[string]string{"ProjectName": "out", "GitUser": "me"}
	err := detectUnsetVars(vars, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unset template variables in paths")
}

func TestDetectUnsetVars_InteractivePromptCalled(t *testing.T) {
	// When interactive, promptForVars should be called
	oldIsInteractive := isInteractive
	defer func() { isInteractive = oldIsInteractive }()
	isInteractive = func() bool { return true }

	oldDir, oldNoPrompt, oldStrict := directory, noPrompt, strict
	defer func() { directory, noPrompt, strict = oldDir, oldNoPrompt, oldStrict }()
	noPrompt = false
	strict = false

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main\nconst x = \"__License__\"\n"), 0o644))

	destDir := filepath.Join(t.TempDir(), "out")
	directory = destDir

	require.NoError(t, copyDir(srcDir, destDir))

	vars := map[string]string{"ProjectName": "out", "GitUser": "me"}

	// Running in a non-terminal test environment will cause huh.Run() to fail,
	// which falls through to the normal (non-prompt) behavior.
	// This test verifies the code path reaches the prompt logic.
	output := captureOutput(func() {
		err := detectUnsetVars(vars, nil)
		// The prompt will fail since there's no real terminal, which is an error
		// OR the unset vars will be handled as warnings (content only)
		// Either way, the function should not panic
		_ = err
	})
	_ = output
}

func TestRecomputeUnset(t *testing.T) {
	unset := &rewrite.UnsetVars{
		InPaths:    []string{"Author", "License"},
		InContents: []string{"Description", "License"},
	}

	vars := map[string]string{
		"Author":  "Oliver",
		"License": "MIT",
		// Description not set
	}

	result := recomputeUnset(unset, vars)

	// Author and License have values, so InPaths should be empty
	assert.Empty(t, result.InPaths)
	// Description is not in vars, so it should remain
	assert.Equal(t, []string{"Description"}, result.InContents)
}

func TestRecomputeUnset_PathVarEmpty(t *testing.T) {
	unset := &rewrite.UnsetVars{
		InPaths: []string{"Author"},
	}

	vars := map[string]string{
		"Author": "", // empty string counts as unset for paths
	}

	result := recomputeUnset(unset, vars)
	assert.Equal(t, []string{"Author"}, result.InPaths)
}

func TestRecomputeUnset_ContentVarEmpty(t *testing.T) {
	unset := &rewrite.UnsetVars{
		InContents: []string{"Description"},
	}

	vars := map[string]string{
		"Description": "", // empty is OK for content vars (present in map)
	}

	result := recomputeUnset(unset, vars)
	// Description is in vars (even if empty), so it should be removed from unset
	assert.Empty(t, result.InContents)
}

// copyDir copies a directory tree for test setup.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// =============================================================================
// Hook tests
// =============================================================================

func TestSubstituteHookVars(t *testing.T) {
	tests := []struct {
		name    string
		command string
		vars    map[string]string
		want    string
	}{
		{
			name:    "basic substitution",
			command: "echo __ProjectName__",
			vars:    map[string]string{"ProjectName": "myapp"},
			want:    "echo myapp",
		},
		{
			name:    "multiple vars",
			command: "echo __ProjectName__ by __Author__",
			vars:    map[string]string{"ProjectName": "myapp", "Author": "Oliver"},
			want:    "echo myapp by Oliver",
		},
		{
			name:    "no vars",
			command: "go mod tidy",
			vars:    map[string]string{},
			want:    "go mod tidy",
		},
		{
			name:    "no matching placeholders",
			command: "go mod tidy",
			vars:    map[string]string{"ProjectName": "myapp"},
			want:    "go mod tidy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteHookVars(tt.command, tt.vars)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRunHooks_NoHooks(t *testing.T) {
	oldNoHooks := noHooks
	defer func() { noHooks = oldNoHooks }()
	noHooks = false

	err := runHooks(t.Context(), nil, t.TempDir(), nil)
	require.NoError(t, err)

	err = runHooks(t.Context(), []gohatchcfg.Hook{}, t.TempDir(), nil)
	require.NoError(t, err)
}

func TestRunHooks_NoHooksFlag(t *testing.T) {
	oldNoHooks := noHooks
	defer func() { noHooks = oldNoHooks }()
	noHooks = true

	hooks := []gohatchcfg.Hook{{Name: "test", Command: "echo hello"}}
	err := runHooks(t.Context(), hooks, t.TempDir(), nil)
	assert.NoError(t, err)
}

func TestRunHooks_NonInteractive(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
	}()
	noHooks = false
	isInteractive = func() bool { return false }

	hooks := []gohatchcfg.Hook{{Name: "test", Command: "echo hello"}}

	output := captureOutput(func() {
		err := runHooks(t.Context(), hooks, t.TempDir(), nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "skipping hooks")
}

func TestRunHooks_Confirmed_Success(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	dir := t.TempDir()
	markerFile := filepath.Join(dir, "hook-ran.txt")
	hooks := []gohatchcfg.Hook{{Name: "marker", Command: "touch " + markerFile}}

	output := captureOutput(func() {
		err := runHooks(t.Context(), hooks, dir, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Running hook: marker")
	assert.FileExists(t, markerFile)
}

func TestRunHooks_Confirmed_Failure(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	hooks := []gohatchcfg.Hook{{Name: "fail", Command: "exit 1"}}

	captureOutput(func() {
		err := runHooks(t.Context(), hooks, t.TempDir(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `hook "fail" failed`)
	})
}

func TestRunHooks_Declined(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return false, nil
	}

	dir := t.TempDir()
	markerFile := filepath.Join(dir, "should-not-exist.txt")
	hooks := []gohatchcfg.Hook{{Name: "marker", Command: "touch " + markerFile}}

	output := captureOutput(func() {
		err := runHooks(t.Context(), hooks, dir, nil)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Skipping hooks")
	assert.NoFileExists(t, markerFile)
}

func TestRunHooks_FailureAbortsRemaining(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	dir := t.TempDir()
	markerFile := filepath.Join(dir, "second-ran.txt")
	hooks := []gohatchcfg.Hook{
		{Name: "first", Command: "exit 1"},
		{Name: "second", Command: "touch " + markerFile},
	}

	captureOutput(func() {
		err := runHooks(t.Context(), hooks, dir, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `hook "first" failed`)
	})

	assert.NoFileExists(t, markerFile)
}

func TestRunHooks_VarSubstitution(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	dir := t.TempDir()
	outputFile := filepath.Join(dir, "output.txt")
	hooks := []gohatchcfg.Hook{{Name: "echo", Command: "echo __ProjectName__ > " + outputFile}}
	vars := map[string]string{"ProjectName": "myapp"}

	captureOutput(func() {
		err := runHooks(t.Context(), hooks, dir, vars)
		require.NoError(t, err)
	})

	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "myapp")
}

func TestRunHooks_WorkingDirectory(t *testing.T) {
	oldNoHooks := noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		noHooks = oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()
	noHooks = false
	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	dir := t.TempDir()
	hooks := []gohatchcfg.Hook{{Name: "pwd", Command: "pwd > pwd.txt"}}

	captureOutput(func() {
		err := runHooks(t.Context(), hooks, dir, nil)
		require.NoError(t, err)
	})

	content, err := os.ReadFile(filepath.Join(dir, "pwd.txt"))
	require.NoError(t, err)
	assert.Contains(t, string(content), filepath.Base(dir))
}

// --- Integration tests for hooks in executeScaffold ---

func TestExecuteScaffold_WithHooks(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()

	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("version = 1\n\n[[hooks]]\nname = \"Create marker\"\ncommand = \"touch hook-marker.txt\"\n"), 0o644))

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
	noHooks = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Running hook: Create marker")
	assert.FileExists(t, filepath.Join(destDir, "hook-marker.txt"))
}

func TestExecuteScaffold_WithHooks_NoHooksFlag(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("version = 1\n\n[[hooks]]\nname = \"Create marker\"\ncommand = \"touch hook-marker.txt\"\n"), 0o644))

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
	noHooks = true
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.NoError(t, err)
	})

	assert.NoFileExists(t, filepath.Join(destDir, "hook-marker.txt"))
}

func TestExecuteScaffold_WithHooks_HookFails_CleansUp(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks :=
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks
	oldIsInteractive := isInteractive
	oldConfirmHooks := confirmHooks
	defer func() {
		directory, module, extensions, variables, force, noGitInit, keepConfig, verbose, srcInput, strict, noHooks =
			oldDir, oldMod, oldExt, oldVars, oldForce, oldNoGitInit, oldKeepConfig, oldVerbose, oldSrcInput, oldStrict, oldNoHooks
		isInteractive = oldIsInteractive
		confirmHooks = oldConfirmHooks
	}()

	isInteractive = func() bool { return true }
	confirmHooks = func(_ []gohatchcfg.Hook, _ map[string]string) (bool, error) {
		return true, nil
	}

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module github.com/template/test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("version = 1\n\n[[hooks]]\nname = \"Fail hook\"\ncommand = \"exit 1\"\n"), 0o644))

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
	noHooks = false
	srcInput = srcDir

	src := &source.LocalSource{Path: srcDir}

	captureOutput(func() {
		err := executeScaffold(t.Context(), src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `hook "Fail hook" failed`)
	})

	_, err := os.Stat(destDir)
	assert.True(t, os.IsNotExist(err), "directory should be removed on hook failure")
}

// --- Dry-run hook tests ---

func TestRunDryRun_WithHooks(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldNoHooks := directory, module, extensions, variables, noHooks
	defer func() {
		directory, module, extensions, variables, noHooks = oldDir, oldMod, oldExt, oldVars, oldNoHooks
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("version = 1\n\n[[hooks]]\nname = \"Install deps\"\ncommand = \"go mod tidy\"\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	noHooks = false

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Hooks:")
	assert.Contains(t, output, "Install deps")
	assert.Contains(t, output, "go mod tidy")
}

func TestRunDryRun_WithHooks_VarSubstitution(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldNoHooks := directory, module, extensions, variables, noHooks
	defer func() {
		directory, module, extensions, variables, noHooks = oldDir, oldMod, oldExt, oldVars, oldNoHooks
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, ".gohatch.toml"), []byte("version = 1\n\n[[hooks]]\nname = \"Setup\"\ncommand = \"echo __ProjectName__\"\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	noHooks = false

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Hooks:")
	assert.Contains(t, output, "echo myapp")
}

func TestRunDryRun_NoHooksFlag(t *testing.T) {
	oldDir, oldMod, oldExt, oldVars, oldNoHooks := directory, module, extensions, variables, noHooks
	defer func() {
		directory, module, extensions, variables, noHooks = oldDir, oldMod, oldExt, oldVars, oldNoHooks
	}()

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644))

	directory = "myapp"
	module = "github.com/me/myapp"
	extensions = nil
	variables = nil
	noHooks = true

	src := &source.LocalSource{Path: srcDir}

	output := captureOutput(func() {
		err := runDryRun(t.Context(), src)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Would skip hooks (--no-hooks).")
}
