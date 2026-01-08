// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package config

// ConfigFile is the name of the configuration file.
const ConfigFile = ".gohatch.toml"

// Config represents the template configuration.
type Config struct {
	Extensions []string `toml:"extensions"`
	Version    int      `toml:"version"`
}
