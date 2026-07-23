package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// SafeWriteFile writes data to a file in the working tree, ensuring path safety.
func SafeWriteFile(repoRoot, relPath string, data []byte, mode uint32) error {
	norm, err := ValidateRelativePath(relPath, repoRoot)
	if err != nil {
		return fmt.Errorf("unsafe restore path %s: %w", relPath, err)
	}

	targetPath := filepath.Join(repoRoot, filepath.FromSlash(norm))
	dir := filepath.Dir(targetPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	perm := os.FileMode(mode)
	if perm == 0 {
		perm = 0644
	}

	if err := os.WriteFile(targetPath, data, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}

	return nil
}

// SafeRemoveFile removes a file from the working tree, ensuring path safety.
func SafeRemoveFile(repoRoot, relPath string) error {
	norm, err := ValidateRelativePath(relPath, repoRoot)
	if err != nil {
		return fmt.Errorf("unsafe removal path %s: %w", relPath, err)
	}

	targetPath := filepath.Join(repoRoot, filepath.FromSlash(norm))
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file %s: %w", targetPath, err)
	}

	// Clean empty parent directories up to repo root
	dir := filepath.Dir(targetPath)
	absRoot, _ := filepath.Abs(repoRoot)
	for dir != absRoot && len(dir) > len(absRoot) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}

	return nil
}
