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

// GitCloner abstracts git cloning operations for testing.
type GitCloner interface {
	PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (Repository, error)
}

// RemoteLister abstracts remote listing operations for testing.
type RemoteLister interface {
	List(o *git.ListOptions) ([]*plumbing.Reference, error)
}

// defaultRemoteLister creates a remote and lists its references.
type defaultRemoteLister struct {
	url string
}

func (d defaultRemoteLister) List(o *git.ListOptions) ([]*plumbing.Reference, error) {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{d.url},
	})
	return remote.List(o)
}

// Repository abstracts git repository operations for testing.
type Repository interface {
	Worktree() (Worktree, error)
}

// Worktree abstracts git worktree operations for testing.
type Worktree interface {
	Checkout(opts *git.CheckoutOptions) error
}

// defaultCloner wraps go-git for production use.
type defaultCloner struct{}

func (defaultCloner) PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (Repository, error) {
	repo, err := git.PlainCloneContext(ctx, path, isBare, o)
	if err != nil {
		return nil, err
	}
	return &gitRepository{repo: repo}, nil
}

type gitRepository struct {
	repo *git.Repository
}

func (r *gitRepository) Worktree() (Worktree, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return nil, err
	}
	return wt, nil
}

// Source represents a template source that can be fetched.
type Source interface {
	Fetch(ctx context.Context, dest string) error
}

// GitSource represents a remote Git repository.
type GitSource struct {
	Cloner  GitCloner    // nil uses default go-git cloner
	Lister  RemoteLister // nil uses default remote lister
	URL     string
	Version string
}

// LocalSource represents a local directory.
type LocalSource struct {
	Path string
}

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

// isCommitHash checks if the version string looks like a Git commit hash.
func isCommitHash(version string) bool {
	if len(version) < 7 || len(version) > 40 {
		return false
	}
	for _, c := range version {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

// refType represents the type of a git reference.
type refType int

const (
	refTypeUnknown refType = iota
	refTypeTag
	refTypeBranch
)

// resolveRefType queries the remote to determine if version is a tag or branch.
func resolveRefType(lister RemoteLister, version string) refType {
	refs, err := lister.List(&git.ListOptions{})
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
	cloner := s.Cloner
	if cloner == nil {
		cloner = defaultCloner{}
	}

	cloneOpts := &git.CloneOptions{
		URL:      s.URL,
		Progress: nil,
	}

	// No version specified: shallow clone of default branch
	if s.Version == "" {
		cloneOpts.Depth = 1
		_, err := cloner.PlainCloneContext(ctx, dest, false, cloneOpts)
		if err != nil {
			return fmt.Errorf("cloning repository: %w", err)
		}
		return removeGitDir(dest)
	}

	// Query remote to determine reference type
	lister := s.Lister
	if lister == nil {
		lister = defaultRemoteLister{url: s.URL}
	}

	refType := resolveRefType(lister, s.Version)

	switch refType {
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
		repo, err := cloner.PlainCloneContext(ctx, dest, false, cloneOpts)
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

		return removeGitDir(dest)
	}

	_, err := cloner.PlainCloneContext(ctx, dest, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	return removeGitDir(dest)
}

// removeGitDir removes the .git directory from the destination.
func removeGitDir(dest string) error {
	gitDir := filepath.Join(dest, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("removing .git directory: %w", err)
	}
	return nil
}

// Fetch copies the local directory to the destination.
func (s *LocalSource) Fetch(_ context.Context, dest string) error {
	return copyDir(s.Path, dest)
}

// copyDir recursively copies a directory.
func copyDir(src, dest string) error {
	// Clean the source path to prevent path traversal
	src = filepath.Clean(src)

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
