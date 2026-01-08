// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)

		content := `version = 1
extensions = ["toml", "yaml", "justfile"]
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		cfg, err := Load(dir)
		require.NoError(t, err)
		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, []string{"toml", "yaml", "justfile"}, cfg.Extensions)
	})

	t.Run("returns empty config when file not found", func(t *testing.T) {
		dir := t.TempDir()

		cfg, err := Load(dir)
		require.NoError(t, err)
		assert.Equal(t, 0, cfg.Version)
		assert.Empty(t, cfg.Extensions)
	})

	t.Run("returns error for invalid TOML", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)

		content := `invalid toml content [`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		_, err := Load(dir)
		assert.Error(t, err)
	})

	t.Run("handles empty extensions", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)

		content := `version = 1
extensions = []
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		cfg, err := Load(dir)
		require.NoError(t, err)
		assert.Equal(t, 1, cfg.Version)
		assert.Empty(t, cfg.Extensions)
	})

	t.Run("defaults version to 1 when not specified", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)

		content := `extensions = ["toml", "yaml"]
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		cfg, err := Load(dir)
		require.NoError(t, err)
		assert.Equal(t, 1, cfg.Version)
		assert.Equal(t, []string{"toml", "yaml"}, cfg.Extensions)
	})
}

func TestExists(t *testing.T) {
	t.Run("returns true when config exists", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)
		require.NoError(t, os.WriteFile(configPath, []byte("version = 1"), 0o644))

		assert.True(t, Exists(dir))
	})

	t.Run("returns false when config does not exist", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, Exists(dir))
	})
}

func TestRemove(t *testing.T) {
	t.Run("removes existing config file", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ConfigFile)
		require.NoError(t, os.WriteFile(configPath, []byte("version = 1"), 0o644))

		err := Remove(dir)
		require.NoError(t, err)

		_, err = os.Stat(configPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("succeeds when file does not exist", func(t *testing.T) {
		dir := t.TempDir()

		err := Remove(dir)
		assert.NoError(t, err)
	})
}
