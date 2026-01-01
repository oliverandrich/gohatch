// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package source

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

// =============================================================================
// Core Interface
// =============================================================================

// Source represents a template source that can be fetched.
type Source interface {
	Fetch(ctx context.Context, dest string) error
}

// =============================================================================
// LocalSource
// =============================================================================

// LocalSource represents a local directory.
type LocalSource struct {
	Path string
}

// Fetch copies the local directory to the destination.
func (s *LocalSource) Fetch(_ context.Context, dest string) error {
	src := filepath.Clean(s.Path)

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o750)
		}

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})
}

// =============================================================================
// GitSource
// =============================================================================

// GitSource represents a remote Git repository.
type GitSource struct {
	URL     string
	Version string
}

// refType represents the type of a git reference.
type refType int

const (
	refTypeUnknown refType = iota
	refTypeTag
	refTypeBranch
)

// resolveRefType queries the remote to determine if version is a tag or branch.
func resolveRefType(url, version string) refType {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{url},
	})

	refs, err := remote.List(&git.ListOptions{})
	if err != nil {
		return refTypeUnknown
	}

	tagRef := plumbing.NewTagReferenceName(version)
	branchRef := plumbing.NewBranchReferenceName(version)

	for _, ref := range refs {
		if ref.Name() == tagRef {
			return refTypeTag
		}
		if ref.Name() == branchRef {
			return refTypeBranch
		}
	}

	return refTypeUnknown
}

// Fetch clones the Git repository to the destination directory.
func (s *GitSource) Fetch(ctx context.Context, dest string) error {
	cloneOpts := &git.CloneOptions{
		URL:      s.URL,
		Progress: nil,
	}

	// No version specified: shallow clone of default branch
	if s.Version == "" {
		cloneOpts.Depth = 1
		_, err := git.PlainCloneContext(ctx, dest, false, cloneOpts)
		if err != nil {
			return fmt.Errorf("cloning repository: %w", err)
		}
		return os.RemoveAll(filepath.Join(dest, ".git"))
	}

	// Query remote to determine reference type
	switch resolveRefType(s.URL, s.Version) {
	case refTypeTag:
		cloneOpts.Depth = 1
		cloneOpts.SingleBranch = true
		cloneOpts.ReferenceName = plumbing.NewTagReferenceName(s.Version)

	case refTypeBranch:
		cloneOpts.Depth = 1
		cloneOpts.SingleBranch = true
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(s.Version)

	case refTypeUnknown:
		// Unknown ref type: assume commit hash, need full clone
		repo, err := git.PlainCloneContext(ctx, dest, false, cloneOpts)
		if err != nil {
			return fmt.Errorf("cloning repository: %w", err)
		}

		worktree, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("getting worktree: %w", err)
		}

		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(s.Version),
		})
		if err != nil {
			return fmt.Errorf("checking out %s: %w", s.Version, err)
		}

		return os.RemoveAll(filepath.Join(dest, ".git"))
	}

	_, err := git.PlainCloneContext(ctx, dest, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	return os.RemoveAll(filepath.Join(dest, ".git"))
}

// =============================================================================
// Parse
// =============================================================================

// Parse analyzes the input string and returns the appropriate Source.
func Parse(input string) (Source, error) {
	path, version := splitVersion(input)

	// Local path: starts with ./, /, or exists as directory
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "/") {
		if version != "" {
			return nil, fmt.Errorf("version specifier not supported for local paths")
		}
		return &LocalSource{Path: path}, nil
	}

	// Check if it's an existing local directory
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if version != "" {
			return nil, fmt.Errorf("version specifier not supported for local paths")
		}
		return &LocalSource{Path: path}, nil
	}

	// Git URL handling
	url := buildGitURL(path)
	return &GitSource{URL: url, Version: version}, nil
}

// splitVersion splits "path@version" into path and version components.
func splitVersion(input string) (path, version string) {
	if idx := strings.LastIndex(input, "@"); idx != -1 {
		return input[:idx], input[idx+1:]
	}
	return input, ""
}

// buildGitURL converts a path to a full HTTPS Git URL.
func buildGitURL(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "https://github.com/" + path
	}

	// Check if first part contains a dot (domain)
	if strings.Contains(parts[0], ".") {
		return "https://" + path
	}

	// Shorthand: user/repo -> github.com/user/repo
	return "https://github.com/" + path
}
