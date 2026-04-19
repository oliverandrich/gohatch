// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRewriteTextFile_SkipsBinary covers the binary-detection short-circuit in
// rewriteTextFile — a file whose content looks binary (contains NUL bytes) is
// left untouched even if the old-module string is present.
func TestRewriteTextFile_SkipsBinary(t *testing.T) {
	tmpDir := t.TempDir()

	binary := []byte("github.com/old/module\x00\x01\x02binary tail")
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "blob.dat"),
		binary,
		0o600,
	))

	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	modified, err := rewriteTextFile(root, "blob.dat", "github.com/old/module", "github.com/new/module")
	require.NoError(t, err)
	require.False(t, modified, "binary file must not be rewritten")

	got, err := os.ReadFile(filepath.Join(tmpDir, "blob.dat"))
	require.NoError(t, err)
	require.Equal(t, binary, got, "binary content must be preserved byte-for-byte")
}

// TestModule_WithExtraExtensions_SkipsBinary exercises the binary-skip path
// through the full Module entry point — templates ship binaries (icons, small
// blobs) that accidentally land on an extra-extension list.
func TestModule_WithExtraExtensions_SkipsBinary(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "go.mod"),
		[]byte("module github.com/old/module\n\ngo 1.21\n"),
		0o600,
	))

	binary := []byte("github.com/old/module\x00payload")
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "data.bin"),
		binary,
		0o600,
	))

	modified, err := Module(tmpDir, "github.com/new/project", []string{"bin"})
	require.NoError(t, err)
	require.Contains(t, modified, "go.mod")
	require.NotContains(t, modified, "data.bin", "binary file must not appear in modified list")

	got, err := os.ReadFile(filepath.Join(tmpDir, "data.bin"))
	require.NoError(t, err)
	require.Equal(t, binary, got, "binary content must survive Module() intact")
}
