package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"minigit/internal/filesystem"
	"minigit/internal/storage"
)

var (
	ErrInvalidBranchName = errors.New("invalid branch name")
	ErrBranchNotFound    = errors.New("branch not found")
	ErrBranchExists      = errors.New("branch already exists")
)

// ValidateBranchName validates rules for branch names.
func ValidateBranchName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidBranchName)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("%w: name cannot contain '..'", ErrInvalidBranchName)
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("%w: name cannot start or end with '/'", ErrInvalidBranchName)
	}
	if strings.Contains(name, "\\") || strings.Contains(name, "//") {
		return fmt.Errorf("%w: name contains invalid path separators", ErrInvalidBranchName)
	}

	for _, r := range name {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%w: name contains control characters or whitespace", ErrInvalidBranchName)
		}
		if r == '~' || r == '^' || r == ':' || r == '?' || r == '*' || r == '[' {
			return fmt.Errorf("%w: name contains invalid character '%c'", ErrInvalidBranchName, r)
		}
	}

	return nil
}

// GetBranchRefPath returns the relative path inside .minigit/refs/heads/ for a branch name.
func GetBranchRefPath(repoRoot, branchName string) (string, error) {
	if err := ValidateBranchName(branchName); err != nil {
		return "", err
	}

	headsDir := filepath.Join(repoRoot, ".minigit", "refs", "heads")
	target := filepath.Join(headsDir, filepath.FromSlash(branchName))

	rel, err := filepath.Rel(headsDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: branch escapes refs directory", ErrInvalidBranchName)
	}

	return target, nil
}

// ReadBranchCommit gets the commit hash for a branch.
func ReadBranchCommit(repoRoot, branchName string) (string, error) {
	refPath, err := GetBranchRefPath(repoRoot, branchName)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(refPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrBranchNotFound, branchName)
		}
		return "", fmt.Errorf("reading branch ref failed: %w", err)
	}

	hash := strings.TrimSpace(string(data))
	if hash == "" {
		return "", fmt.Errorf("%w: branch ref is empty", ErrBranchNotFound)
	}

	return hash, nil
}

// WriteBranchCommit creates or updates a branch ref atomically.
func WriteBranchCommit(repoRoot, branchName, commitHash string) error {
	refPath, err := GetBranchRefPath(repoRoot, branchName)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("%s\n", commitHash)
	return storage.WriteFileAtomic(refPath, []byte(content), 0644)
}

// CreateBranch creates a new branch pointing to a commit hash if it doesn't already exist.
func CreateBranch(repoRoot, branchName, commitHash string) error {
	refPath, err := GetBranchRefPath(repoRoot, branchName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(refPath); err == nil {
		return fmt.Errorf("%w: %s", ErrBranchExists, branchName)
	}

	return WriteBranchCommit(repoRoot, branchName, commitHash)
}

// ListBranches returns sorted list of all branch names.
func ListBranches(repoRoot string) ([]string, error) {
	headsDir := filepath.Join(repoRoot, ".minigit", "refs", "heads")
	var branches []string

	err := filepath.Walk(headsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			rel, err := filepath.Rel(headsDir, path)
			if err == nil {
				branches = append(branches, filesystem.NormalizePath(rel))
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	sort.Strings(branches)
	return branches, nil
}

// ListBranches returns sorted list of all branch names.
func (r *Repository) ListBranches() ([]string, error) {
	return ListBranches(r.Root)
}

// CreateBranch creates a new branch pointing to a commit hash if it doesn't already exist.
func (r *Repository) CreateBranch(branchName, commitHash string) error {
	return CreateBranch(r.Root, branchName, commitHash)
}
