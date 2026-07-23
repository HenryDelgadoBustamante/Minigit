package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileAtomic writes data to a file atomically by writing to a temporary file, syncing, and renaming.
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".minigit-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpName := tmpFile.Name()
	cleanedUp := false

	defer func() {
		if !cleanedUp {
			tmpFile.Close()
			os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to temporary file: %w", err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to synchronize temporary file to disk: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to atomically replace destination file %s: %w", filename, err)
	}

	// Sync directory to ensure rename is persisted
	dirFile, err := os.Open(dir)
	if err == nil {
		dirFile.Sync()
		dirFile.Close()
	}

	cleanedUp = true
	return nil
}

// CleanupTempFiles removes abandoned temporary files matching the .minigit-tmp-* pattern.
func CleanupTempFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".minigit-tmp-") {
			fullPath := filepath.Join(dir, entry.Name())
			if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
				return cleaned, fmt.Errorf("failed to remove stale temp file %s: %w", fullPath, err)
			}
			cleaned++
		}
	}

	return cleaned, nil
}

// CleanupTempFilesRecursive walks the directory tree and removes all .minigit-tmp-* files.
func CleanupTempFilesRecursive(rootDir string) (int, error) {
	totalCleaned := 0

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (e.g., permission denied)
		}
		if info.IsDir() {
			cleaned, err := CleanupTempFiles(path)
			if err != nil {
				return err
			}
			totalCleaned += cleaned
		}
		return nil
	})

	return totalCleaned, err
}
