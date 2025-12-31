// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package source

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// mockCloner is a mock implementation of GitCloner for testing.
type mockCloner struct {
	mock.Mock
}

func (m *mockCloner) PlainCloneContext(ctx context.Context, path string, isBare bool, o *git.CloneOptions) (Repository, error) {
	args := m.Called(ctx, path, isBare, o)
	repo := args.Get(0)
	if repo == nil {
		return nil, args.Error(1)
	}
	return repo.(Repository), args.Error(1)
}

// mockRepository is a mock implementation of Repository for testing.
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Worktree() (Worktree, error) {
	args := m.Called()
	wt := args.Get(0)
	if wt == nil {
		return nil, args.Error(1)
	}
	return wt.(Worktree), args.Error(1)
}

// mockWorktree is a mock implementation of Worktree for testing.
type mockWorktree struct {
	mock.Mock
}

func (m *mockWorktree) Checkout(opts *git.CheckoutOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

// mockLister is a mock implementation of RemoteLister for testing.
type mockLister struct {
	mock.Mock
}

func (m *mockLister) List(o *git.ListOptions) ([]*plumbing.Reference, error) {
	args := m.Called(o)
	refs := args.Get(0)
	if refs == nil {
		return nil, args.Error(1)
	}
	return refs.([]*plumbing.Reference), args.Error(1)
}

// Helper to create mock refs
func mockRef(refPath string) *plumbing.Reference {
	return plumbing.NewReferenceFromStrings(refPath, "0000000000000000000000000000000000000000")
}

func TestGitSourceFetch_DefaultBranch_Success(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	repo := new(mockRepository)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.MatchedBy(func(o *git.CloneOptions) bool {
		return o.Depth == 1 && o.URL == "https://github.com/user/repo"
	})).Return(repo, nil)

	gs := &GitSource{
		URL:    "https://github.com/user/repo",
		Cloner: cloner,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.NoError(t, err)
	cloner.AssertExpectations(t)
}

func TestGitSourceFetch_DefaultBranch_CloneError(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.Anything).
		Return(nil, errors.New("network error"))

	gs := &GitSource{
		URL:    "https://github.com/user/repo",
		Cloner: cloner,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloning repository")
	assert.Contains(t, err.Error(), "network error")
	cloner.AssertExpectations(t)
}

func TestGitSourceFetch_Tag_Success(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)

	// Lister returns v1.0.0 as a tag
	lister.On("List", mock.Anything).Return([]*plumbing.Reference{
		mockRef("refs/tags/v1.0.0"),
		mockRef("refs/heads/main"),
	}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.MatchedBy(func(o *git.CloneOptions) bool {
		return o.Depth == 1 && o.SingleBranch && o.ReferenceName.String() == "refs/tags/v1.0.0"
	})).Return(repo, nil)

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "v1.0.0",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.NoError(t, err)
	cloner.AssertExpectations(t)
	lister.AssertExpectations(t)
}

func TestGitSourceFetch_Tag_CloneError(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)

	lister.On("List", mock.Anything).Return([]*plumbing.Reference{
		mockRef("refs/tags/v1.0.0"),
	}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.Anything).
		Return(nil, errors.New("tag not found"))

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "v1.0.0",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloning repository")
	cloner.AssertExpectations(t)
}

func TestGitSourceFetch_Branch_Success(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)

	// Lister returns main as a branch
	lister.On("List", mock.Anything).Return([]*plumbing.Reference{
		mockRef("refs/tags/v1.0.0"),
		mockRef("refs/heads/main"),
	}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.MatchedBy(func(o *git.CloneOptions) bool {
		return o.Depth == 1 && o.SingleBranch && o.ReferenceName.String() == "refs/heads/main"
	})).Return(repo, nil)

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "main",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.NoError(t, err)
	cloner.AssertExpectations(t)
	lister.AssertExpectations(t)
}

func TestGitSourceFetch_Branch_FeatureBranch(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)

	lister.On("List", mock.Anything).Return([]*plumbing.Reference{
		mockRef("refs/heads/feature/new-feature"),
	}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.MatchedBy(func(o *git.CloneOptions) bool {
		return o.Depth == 1 && o.SingleBranch && o.ReferenceName.String() == "refs/heads/feature/new-feature"
	})).Return(repo, nil)

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "feature/new-feature",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.NoError(t, err)
	cloner.AssertExpectations(t)
	lister.AssertExpectations(t)
}

func TestGitSourceFetch_CommitHash_Success(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)
	worktree := new(mockWorktree)

	// Lister returns refs that don't match, so it falls back to commit
	lister.On("List", mock.Anything).Return([]*plumbing.Reference{
		mockRef("refs/tags/v1.0.0"),
		mockRef("refs/heads/main"),
	}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.MatchedBy(func(o *git.CloneOptions) bool {
		return o.Depth == 0 // full clone for commit hash
	})).Return(repo, nil)

	repo.On("Worktree").Return(worktree, nil)
	worktree.On("Checkout", mock.Anything).Return(nil)

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "abc1234def",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.NoError(t, err)
	cloner.AssertExpectations(t)
	lister.AssertExpectations(t)
	repo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}

func TestGitSourceFetch_CommitHash_CloneError(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)

	lister.On("List", mock.Anything).Return([]*plumbing.Reference{}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.Anything).
		Return(nil, errors.New("clone failed"))

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "abc1234def",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cloning repository")
	cloner.AssertExpectations(t)
}

func TestGitSourceFetch_CommitHash_WorktreeError(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)

	lister.On("List", mock.Anything).Return([]*plumbing.Reference{}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.Anything).Return(repo, nil)
	repo.On("Worktree").Return(nil, errors.New("worktree error"))

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "abc1234def",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting worktree")
	cloner.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestGitSourceFetch_CommitHash_CheckoutError(t *testing.T) {
	destDir := t.TempDir() + "/dest"
	cloner := new(mockCloner)
	lister := new(mockLister)
	repo := new(mockRepository)
	worktree := new(mockWorktree)

	lister.On("List", mock.Anything).Return([]*plumbing.Reference{}, nil)

	cloner.On("PlainCloneContext", mock.Anything, destDir, false, mock.Anything).Return(repo, nil)
	repo.On("Worktree").Return(worktree, nil)
	worktree.On("Checkout", mock.Anything).Return(errors.New("commit not found"))

	gs := &GitSource{
		URL:     "https://github.com/user/repo",
		Version: "abc1234def",
		Cloner:  cloner,
		Lister:  lister,
	}

	err := gs.Fetch(context.Background(), destDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checking out")
	cloner.AssertExpectations(t)
	repo.AssertExpectations(t)
	worktree.AssertExpectations(t)
}
