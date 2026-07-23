package filesystem

import (
	"os"
	"path/filepath"
	"sort"
)

type FileItem struct {
	RelPath string
	AbsPath string
	Info    os.FileInfo
}

type IgnoreChecker interface {
	IsIgnored(relPath string, isDir bool) bool
}

// WalkWorkingTree recursively collects all unignored files in the repository.
func WalkWorkingTree(repoRoot string, ignoreChecker IgnoreChecker) ([]FileItem, error) {
	var items []FileItem
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if path == absRoot {
			return nil
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		norm := NormalizePath(rel)

		if ignoreChecker != nil && ignoreChecker.IsIgnored(norm, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Validate symlinks if any
		if info.Mode()&os.ModeSymlink != 0 {
			if err := ValidateSymlink(path, absRoot); err != nil {
				return err // safe reject unsafe symlinks
			}
		}

		if !info.IsDir() {
			items = append(items, FileItem{
				RelPath: norm,
				AbsPath: path,
				Info:    info,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].RelPath < items[j].RelPath
	})

	return items, nil
}
