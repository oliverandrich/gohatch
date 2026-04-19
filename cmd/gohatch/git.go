// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// validateDirectory checks that the target directory doesn't exist or is empty.
func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil // Directory doesn't exist, OK
	}
	if err != nil {
		return fmt.Errorf("checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", dir)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("directory %s is not empty", dir)
	}

	return nil
}

// initGitRepo initializes a git repository and creates an initial commit.
func initGitRepo(dir string) error {
	repo, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName("main"),
		},
	})
	if err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	// Add all files
	if err = worktree.AddGlob("."); err != nil {
		return fmt.Errorf("staging files: %w", err)
	}

	// Create initial commit
	_, err = worktree.Commit("Initial commit.", &git.CommitOptions{
		Author: getGitAuthor(),
	})
	if err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	fmt.Println("\nInitialized git repository with initial commit")
	return nil
}

// getGitAuthor reads the git author from the user's global git config.
// Falls back to gohatch defaults if not configured.
func getGitAuthor() *object.Signature {
	cfg, err := config.LoadConfig(config.GlobalScope)
	if err == nil && cfg.User.Name != "" && cfg.User.Email != "" {
		return &object.Signature{
			Name:  cfg.User.Name,
			Email: cfg.User.Email,
			When:  time.Now(),
		}
	}
	return &object.Signature{
		Name:  "gohatch",
		Email: "gohatch@localhost",
		When:  time.Now(),
	}
}
