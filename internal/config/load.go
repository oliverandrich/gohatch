// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load reads the config from a .gohatch.toml file in the given directory.
// Returns an empty Config if no config file exists.
// Version defaults to 1 if not specified.
func Load(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ConfigFile)

	data, err := os.ReadFile(configPath) //nolint:gosec // configPath is constructed from trusted directory
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Default version to 1 if not specified
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	return &cfg, nil
}

// Exists checks if a config file exists in the given directory.
func Exists(dir string) bool {
	configPath := filepath.Join(dir, ConfigFile)
	_, err := os.Stat(configPath)
	return err == nil
}
