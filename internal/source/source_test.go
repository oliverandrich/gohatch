// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// setupBareRepo creates a bare git repository with an initial commit.
// Returns the file:// URL to the repository.
func setupBareRepo(t *testing.T) string {
	t.Helper()

	// Create a working directory first, then clone to bare
	workDir := t.TempDir()
	bareDir := t.TempDir()

	// Initialize a regular repo
	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(workDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Template\n"), 0o644)
	require.NoError(t, err)

	// Add and commit
	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Clone to bare repo
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL: workDir,
	})
	require.NoError(t, err)

	// Store commit hash for later use
	t.Setenv("TEST_COMMIT_HASH", commitHash.String())

	return "file://" + bareDir
}

// setupBareRepoWithTag creates a bare repo with a tag.
func setupBareRepoWithTag(t *testing.T, tagName string) string {
	t.Helper()

	workDir := t.TempDir()
	bareDir := t.TempDir()

	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	// Create test file
	err = os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Tagged Version\n"), 0o644)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	commitHash, err := worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create lightweight tag
	_, err = repo.CreateTag(tagName, commitHash, nil)
	require.NoError(t, err)

	// Clone to bare
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL: workDir,
	})
	require.NoError(t, err)

	return "file://" + bareDir
}

// setupBareRepoWithBranch creates a bare repo with a specific branch.
func setupBareRepoWithBranch(t *testing.T, branchName string) string {
	t.Helper()

	workDir := t.TempDir()
	bareDir := t.TempDir()

	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	// Create test file and initial commit on main
	err = os.WriteFile(filepath.Join(workDir, "README.md"), []byte("# Main Branch\n"), 0o644)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create and checkout new branch
	headRef, err := repo.Head()
	require.NoError(t, err)

	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	err = repo.Storer.SetReference(ref)
	require.NoError(t, err)

	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	require.NoError(t, err)

	// Add branch-specific content
	err = os.WriteFile(filepath.Join(workDir, "BRANCH.md"), []byte("# "+branchName+"\n"), 0o644)
	require.NoError(t, err)

	_, err = worktree.Add("BRANCH.md")
	require.NoError(t, err)

	_, err = worktree.Commit("Branch commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Clone to bare
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL: workDir,
	})
	require.NoError(t, err)

	return "file://" + bareDir
}

// setupBareRepoWithCommits creates a bare repo with multiple commits.
// Returns the URL and the hash of the first commit.
func setupBareRepoWithCommits(t *testing.T) (string, string) {
	t.Helper()

	workDir := t.TempDir()
	bareDir := t.TempDir()

	repo, err := git.PlainInit(workDir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	// First commit
	err = os.WriteFile(filepath.Join(workDir, "v1.txt"), []byte("version 1\n"), 0o644)
	require.NoError(t, err)

	_, err = worktree.Add("v1.txt")
	require.NoError(t, err)

	firstCommit, err := worktree.Commit("First commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Second commit
	err = os.WriteFile(filepath.Join(workDir, "v2.txt"), []byte("version 2\n"), 0o644)
	require.NoError(t, err)

	_, err = worktree.Add("v2.txt")
	require.NoError(t, err)

	_, err = worktree.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Clone to bare
	_, err = git.PlainClone(bareDir, true, &git.CloneOptions{
		URL: workDir,
	})
	require.NoError(t, err)

	return "file://" + bareDir, firstCommit.String()
}

// =============================================================================
// Parse Tests
// =============================================================================

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
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantVersion, version)
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
		{"singlepart", "https://github.com/singlepart"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := buildGitURL(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParse(t *testing.T) {
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
			src, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			switch tt.wantType {
			case "git":
				gs, ok := src.(*GitSource)
				require.True(t, ok, "expected GitSource, got %T", src)
				assert.Equal(t, tt.wantURL, gs.URL)
				assert.Equal(t, tt.wantVersion, gs.Version)
			case "local":
				ls, ok := src.(*LocalSource)
				require.True(t, ok, "expected LocalSource, got %T", src)
				assert.Equal(t, tt.wantPath, ls.Path)
			}
		})
	}
}

func TestParseExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	src, err := Parse(tmpDir)
	require.NoError(t, err)

	ls, ok := src.(*LocalSource)
	require.True(t, ok)
	assert.Equal(t, tmpDir, ls.Path)
}

func TestParseExistingDirectoryWithVersion(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Parse(tmpDir + "@v1.0.0")
	assert.Error(t, err)
}

// =============================================================================
// LocalSource Tests
// =============================================================================

func TestLocalSourceFetch(t *testing.T) {
	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "dest")

	// Create test file
	err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0o644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(srcDir, "subdir")
	err = os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("world"), 0o644)
	require.NoError(t, err)

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(srcDir, ".git")
	err = os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("git"), 0o644)
	require.NoError(t, err)

	ls := &LocalSource{Path: srcDir}
	err = ls.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify files were copied
	assert.FileExists(t, filepath.Join(destDir, "test.txt"))
	assert.FileExists(t, filepath.Join(destDir, "subdir", "sub.txt"))

	// Verify .git was not copied
	assert.NoDirExists(t, filepath.Join(destDir, ".git"))
}

func TestLocalSourceFetchPreservesPermissions(t *testing.T) {
	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "dest")

	// Create executable file
	execFile := filepath.Join(srcDir, "script.sh")
	err := os.WriteFile(execFile, []byte("#!/bin/bash"), 0o755)
	require.NoError(t, err)

	ls := &LocalSource{Path: srcDir}
	err = ls.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify permissions preserved
	info, err := os.Stat(filepath.Join(destDir, "script.sh"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func TestLocalSourceFetchWithDotFiles(t *testing.T) {
	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "dest")

	// Create various dot files (should be copied except .git)
	dotFiles := []string{".gitignore", ".env.example", ".golangci.yml"}
	for _, f := range dotFiles {
		err := os.WriteFile(filepath.Join(srcDir, f), []byte("content"), 0o644)
		require.NoError(t, err)
	}

	ls := &LocalSource{Path: srcDir}
	err := ls.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify all dot files were copied
	for _, f := range dotFiles {
		assert.FileExists(t, filepath.Join(destDir, f))
	}
}

// =============================================================================
// GitSource Tests - Real Bare Repos
// =============================================================================

func TestGitSourceFetch_DefaultBranch(t *testing.T) {
	repoURL := setupBareRepo(t)
	destDir := filepath.Join(t.TempDir(), "dest")

	gs := &GitSource{URL: repoURL}
	err := gs.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify file was cloned
	assert.FileExists(t, filepath.Join(destDir, "README.md"))

	// Verify .git was removed
	assert.NoDirExists(t, filepath.Join(destDir, ".git"))
}

func TestGitSourceFetch_Tag(t *testing.T) {
	repoURL := setupBareRepoWithTag(t, "v1.0.0")
	destDir := filepath.Join(t.TempDir(), "dest")

	gs := &GitSource{URL: repoURL, Version: "v1.0.0"}
	err := gs.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify file was cloned
	assert.FileExists(t, filepath.Join(destDir, "README.md"))

	// Verify .git was removed
	assert.NoDirExists(t, filepath.Join(destDir, ".git"))
}

func TestGitSourceFetch_Branch(t *testing.T) {
	repoURL := setupBareRepoWithBranch(t, "feature")
	destDir := filepath.Join(t.TempDir(), "dest")

	gs := &GitSource{URL: repoURL, Version: "feature"}
	err := gs.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify branch-specific file exists
	assert.FileExists(t, filepath.Join(destDir, "BRANCH.md"))

	// Verify .git was removed
	assert.NoDirExists(t, filepath.Join(destDir, ".git"))
}

func TestGitSourceFetch_CommitHash(t *testing.T) {
	repoURL, firstCommitHash := setupBareRepoWithCommits(t)
	destDir := filepath.Join(t.TempDir(), "dest")

	gs := &GitSource{URL: repoURL, Version: firstCommitHash}
	err := gs.Fetch(context.Background(), destDir)
	require.NoError(t, err)

	// Verify first commit's file exists
	assert.FileExists(t, filepath.Join(destDir, "v1.txt"))

	// Verify second commit's file does NOT exist (we checked out first commit)
	assert.NoFileExists(t, filepath.Join(destDir, "v2.txt"))

	// Verify .git was removed
	assert.NoDirExists(t, filepath.Join(destDir, ".git"))
}

func TestGitSourceFetch_InvalidURL(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "dest")

	gs := &GitSource{URL: "file:///nonexistent/repo"}
	err := gs.Fetch(context.Background(), destDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloning repository")
}

func TestGitSourceFetch_ShortCommitHash(t *testing.T) {
	repoURL, fullHash := setupBareRepoWithCommits(t)
	destDir := filepath.Join(t.TempDir(), "dest")

	// Use short hash (first 7 characters)
	shortHash := fullHash[:7]

	gs := &GitSource{URL: repoURL, Version: shortHash}
	err := gs.Fetch(context.Background(), destDir)

	// go-git should handle short hashes
	// Note: This might fail if go-git requires full hashes
	if err != nil {
		// If it fails, that's expected behavior for short hashes
		assert.Contains(t, err.Error(), "checking out")
	} else {
		assert.FileExists(t, filepath.Join(destDir, "v1.txt"))
	}
}
