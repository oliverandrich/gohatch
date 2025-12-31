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
