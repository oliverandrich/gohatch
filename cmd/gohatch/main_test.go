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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

func TestValidateDirectory_IsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("content"), 0o644))

	err := validateDirectory(file)
	assert.Error(t, err)
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

	assert.Contains(t, output, "Extensions: [toml yaml]")
	assert.Contains(t, output, "files with specified extensions")
}

func TestParseVariables_DefaultProjectName(t *testing.T) {
	vars := parseVariables(nil, "myapp")

	assert.Equal(t, "myapp", vars["ProjectName"])
	assert.Len(t, vars, 1)
}

func TestParseVariables_WithVars(t *testing.T) {
	input := []string{"Author=Oliver Andrich", "License=MIT"}
	vars := parseVariables(input, "myapp")

	assert.Equal(t, "myapp", vars["ProjectName"])
	assert.Equal(t, "Oliver Andrich", vars["Author"])
	assert.Equal(t, "MIT", vars["License"])
	assert.Len(t, vars, 3)
}

func TestParseVariables_OverrideProjectName(t *testing.T) {
	input := []string{"ProjectName=CustomName"}
	vars := parseVariables(input, "myapp")

	assert.Equal(t, "CustomName", vars["ProjectName"])
	assert.Len(t, vars, 1)
}

func TestParseVariables_ValueWithEquals(t *testing.T) {
	// strings.Cut splits only on the first =, so value keeps the rest
	input := []string{"Equation=a=b+c"}
	vars := parseVariables(input, "myapp")

	assert.Equal(t, "a=b+c", vars["Equation"])
}

func TestParseVariables_InvalidEntry(t *testing.T) {
	input := []string{"NoEqualsSign"}
	vars := parseVariables(input, "myapp")

	// Should only have default ProjectName
	assert.Len(t, vars, 1)
	assert.Equal(t, "myapp", vars["ProjectName"])
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
