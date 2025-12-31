// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package source

import (
	"context"
	"os"
	"testing"
)

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"abc1234", true},
		{"ABC1234", true},
		{"abcdef1234567890abcdef1234567890abcdef12", true}, // 40 chars
		{"abc123", false},  // too short (6 chars)
		{"v1.0.0", false},  // contains non-hex
		{"latest", false},  // not hex
		{"main", false},    // not hex
		{"abc12g4", false}, // contains 'g'
		{"", false},        // empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isCommitHash(tt.input)
			if got != tt.want {
				t.Errorf("isCommitHash(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantPath    string
		wantVersion string
	}{
		{"user/repo", "user/repo", ""},
		{"user/repo@v1.0.0", "user/repo", "v1.0.0"},
		{"github.com/user/repo@v2.1.0", "github.com/user/repo", "v2.1.0"},
		{"github.com/user/repo", "github.com/user/repo", ""},
		{"user/repo@latest", "user/repo", "latest"},
		{"user/repo@abc1234", "user/repo", "abc1234"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, version := splitVersion(tt.input)
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestBuildGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user/repo", "https://github.com/user/repo"},
		{"github.com/user/repo", "https://github.com/user/repo"},
		{"codeberg.org/user/repo", "https://codeberg.org/user/repo"},
		{"gitlab.com/user/repo", "https://gitlab.com/user/repo"},
		{"user/repo/subdir", "https://github.com/user/repo/subdir"},
		{"singlepart", "https://github.com/singlepart"}, // no slash
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := buildGitURL(tt.input)
			if got != tt.want {
				t.Errorf("buildGitURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	// Create a temporary directory for local path tests
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		input       string
		wantType    string
		wantURL     string
		wantPath    string
		wantVersion string
		wantErr     bool
	}{
		{
			name:     "shorthand without version",
			input:    "user/repo",
			wantType: "git",
			wantURL:  "https://github.com/user/repo",
		},
		{
			name:        "shorthand with version",
			input:       "user/repo@v1.0.0",
			wantType:    "git",
			wantURL:     "https://github.com/user/repo",
			wantVersion: "v1.0.0",
		},
		{
			name:     "full github URL",
			input:    "github.com/user/repo",
			wantType: "git",
			wantURL:  "https://github.com/user/repo",
		},
		{
			name:        "full github URL with version",
			input:       "github.com/user/repo@v2.0.0",
			wantType:    "git",
			wantURL:     "https://github.com/user/repo",
			wantVersion: "v2.0.0",
		},
		{
			name:     "codeberg URL",
			input:    "codeberg.org/user/repo",
			wantType: "git",
			wantURL:  "https://codeberg.org/user/repo",
		},
		{
			name:        "shorthand with commit hash",
			input:       "user/repo@abc1234def",
			wantType:    "git",
			wantURL:     "https://github.com/user/repo",
			wantVersion: "abc1234def",
		},
		{
			name:     "relative path",
			input:    "./some/path",
			wantType: "local",
			wantPath: "./some/path",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			wantType: "local",
			wantPath: "/absolute/path",
		},
		{
			name:    "relative path with version error",
			input:   "./some/path@v1.0.0",
			wantErr: true,
		},
		{
			name:    "absolute path with version error",
			input:   "/absolute/path@v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For existing directory test
			if tt.name == "existing directory" {
				tt.input = tmpDir
				tt.wantPath = tmpDir
			}

			src, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			switch tt.wantType {
			case "git":
				gs, ok := src.(*GitSource)
				if !ok {
					t.Errorf("expected GitSource, got %T", src)
					return
				}
				if gs.URL != tt.wantURL {
					t.Errorf("URL = %q, want %q", gs.URL, tt.wantURL)
				}
				if gs.Version != tt.wantVersion {
					t.Errorf("Version = %q, want %q", gs.Version, tt.wantVersion)
				}
			case "local":
				ls, ok := src.(*LocalSource)
				if !ok {
					t.Errorf("expected LocalSource, got %T", src)
					return
				}
				if ls.Path != tt.wantPath {
					t.Errorf("Path = %q, want %q", ls.Path, tt.wantPath)
				}
			}
		})
	}
}

func TestParseExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	src, err := Parse(tmpDir)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	ls, ok := src.(*LocalSource)
	if !ok {
		t.Fatalf("expected LocalSource, got %T", src)
	}
	if ls.Path != tmpDir {
		t.Errorf("Path = %q, want %q", ls.Path, tmpDir)
	}
}

func TestParseExistingDirectoryWithVersion(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Parse(tmpDir + "@v1.0.0")
	if err == nil {
		t.Error("Parse() should error for existing directory with version")
	}
}

func TestLocalSourceFetchPreservesPermissions(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir() + "/dest"

	// Create executable file
	execFile := srcDir + "/script.sh"
	if err := os.WriteFile(execFile, []byte("#!/bin/bash"), 0o755); err != nil {
		t.Fatal(err)
	}

	ls := &LocalSource{Path: srcDir}
	if err := ls.Fetch(context.Background(), destDir); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify permissions preserved
	info, err := os.Stat(destDir + "/script.sh")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("permissions = %o, want 0755", info.Mode().Perm())
	}
}

func TestLocalSourceFetchWithDotFiles(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir() + "/dest"

	// Create various dot files (should be copied except .git)
	dotFiles := []string{".gitignore", ".env.example", ".golangci.yml"}
	for _, f := range dotFiles {
		if err := os.WriteFile(srcDir+"/"+f, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ls := &LocalSource{Path: srcDir}
	if err := ls.Fetch(context.Background(), destDir); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify all dot files were copied
	for _, f := range dotFiles {
		if _, err := os.Stat(destDir + "/" + f); err != nil {
			t.Errorf("%s not copied: %v", f, err)
		}
	}
}

func TestLocalSourceFetch(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir()
	destDir := t.TempDir() + "/dest"

	// Create test file
	testFile := srcDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory
	subDir := srcDir + "/subdir"
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subFile := subDir + "/sub.txt"
	if err := os.WriteFile(subFile, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .git directory (should be skipped)
	gitDir := srcDir + "/.git"
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitFile := gitDir + "/config"
	if err := os.WriteFile(gitFile, []byte("git"), 0o644); err != nil {
		t.Fatal(err)
	}

	ls := &LocalSource{Path: srcDir}
	if err := ls.Fetch(context.Background(), destDir); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify files were copied
	if _, err := os.Stat(destDir + "/test.txt"); err != nil {
		t.Errorf("test.txt not copied: %v", err)
	}
	if _, err := os.Stat(destDir + "/subdir/sub.txt"); err != nil {
		t.Errorf("subdir/sub.txt not copied: %v", err)
	}

	// Verify .git was not copied
	if _, err := os.Stat(destDir + "/.git"); err == nil {
		t.Error(".git directory should not be copied")
	}
}
