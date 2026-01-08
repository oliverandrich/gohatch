// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package config

import (
	"errors"
	"os"
	"path/filepath"
)

// Remove deletes the config file from the given directory.
// Returns nil if the file doesn't exist.
func Remove(dir string) error {
	configPath := filepath.Join(dir, ConfigFile)

	err := os.Remove(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
