package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrAbsolutePath  = errors.New("absolute paths are forbidden")
	ErrPathTraversal = errors.New("path traversal (..) is forbidden")
	ErrInternalPath  = errors.New("accessing internal repository paths (.minigit/.git) is forbidden")
	ErrNullByte      = errors.New("null bytes in path are forbidden")
	ErrOutsideRepo   = errors.New("path escapes repository root")
	ErrUnsafeSymlink = errors.New("symlink points outside repository root")
)

// NormalizePath converts backslashes to forward slashes and cleans the path.
func NormalizePath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.ReplaceAll(cleaned, "\\", "/")
}

// ValidateRelativePath verifies that a path is safe to operate on relative to the repo root.
func ValidateRelativePath(relPath string, repoRoot string) (string, error) {
	if strings.Contains(relPath, "\x00") {
		return "", ErrNullByte
	}

	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "/") || (len(relPath) > 1 && relPath[1] == ':') {
		return "", ErrAbsolutePath
	}

	norm := NormalizePath(relPath)

	if norm == ".." || strings.HasPrefix(norm, "../") || strings.Contains(norm, "/../") {
		return "", ErrPathTraversal
	}

	if norm == ".minigit" || strings.HasPrefix(norm, ".minigit/") || norm == ".git" || strings.HasPrefix(norm, ".git/") {
		return "", ErrInternalPath
	}

	// Verify target path absolute position relative to repo root
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("invalid repository root: %w", err)
	}

	targetAbs := filepath.Clean(filepath.Join(absRoot, norm))
	relToRoot, err := filepath.Rel(absRoot, targetAbs)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", ErrOutsideRepo
	}

	return norm, nil
}

// ValidateSymlink checks if a symlink target stays within the repo root directory.
func ValidateSymlink(symlinkPath string, repoRoot string) error {
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}

	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink target: %w", err)
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}

	var resolved string
	if filepath.IsAbs(target) {
		resolved = filepath.Clean(target)
	} else {
		resolved = filepath.Clean(filepath.Join(filepath.Dir(symlinkPath), target))
	}

	rel, err := filepath.Rel(absRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: symlink %s points to %s", ErrUnsafeSymlink, symlinkPath, target)
	}

	return nil
}

// GetFileMode returns standard file permissions or executable bit (0755 or 0644).
func GetFileMode(info os.FileInfo) uint32 {
	if info.Mode().IsDir() {
		return 0755
	}
	if info.Mode()&0111 != 0 {
		return 0755
	}
	return 0644
}
