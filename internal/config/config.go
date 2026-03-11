// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package config

// ConfigFile is the name of the configuration file.
const ConfigFile = ".gohatch.toml"

// Hook represents a post-generation hook command.
type Hook struct {
	Name    string `toml:"name"`
	Command string `toml:"command"`
}

// Config represents the template configuration.
type Config struct {
	Extensions []string `toml:"extensions"`
	Hooks      []Hook   `toml:"hooks"`
	Version    int      `toml:"version"`
}
