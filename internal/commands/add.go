package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"minigit/internal/filesystem"
	"minigit/internal/object"
	"minigit/internal/repository"
)

// RunAdd stages specified files or directories.
func RunAdd(repo *repository.Repository, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("nothing specified, nothing added")
	}

	lock, err := repository.AcquireLock(repository.GetIndexPath(repo.Root))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	idx, err := repository.ReadIndex(repo.Root)
	if err != nil {
		return err
	}

	for _, p := range paths {
		if err := addPath(repo, idx, p); err != nil {
			return err
		}
	}

	return repository.WriteIndex(repo.Root, idx)
}

func addPath(repo *repository.Repository, idx *repository.Index, argPath string) error {
	if argPath == "." {
		return addAllRecursive(repo, idx)
	}

	normPath, err := filesystem.ValidateRelativePath(argPath, repo.Root)
	if err != nil {
		return fmt.Errorf("invalid path '%s': %w", argPath, err)
	}

	absPath := filepath.Join(repo.Root, filepath.FromSlash(normPath))
	info, err := os.Lstat(absPath)

	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted on disk. If it exists in index, stage deletion
			if _, exists := idx.Entries[normPath]; exists {
				idx.RemoveEntry(normPath)
				return nil
			}
			return fmt.Errorf("pathspec '%s' did not match any files", argPath)
		}
		return fmt.Errorf("failed to stat '%s': %w", argPath, err)
	}

	if repo.Ignore.IsIgnored(normPath, info.IsDir()) {
		return nil
	}

	if info.IsDir() {
		return addDirectoryRecursive(repo, idx, normPath)
	}

	return stageSingleFile(repo, idx, normPath, absPath, info)
}

func addAllRecursive(repo *repository.Repository, idx *repository.Index) error {
	items, err := filesystem.WalkWorkingTree(repo.Root, repo.Ignore)
	if err != nil {
		return fmt.Errorf("failed to walk repository tree: %w", err)
	}

	workTreePaths := make(map[string]bool)
	for _, item := range items {
		workTreePaths[item.RelPath] = true
		if err := stageSingleFile(repo, idx, item.RelPath, item.AbsPath, item.Info); err != nil {
			return err
		}
	}

	// Detect deleted tracked files in working directory
	for trackedPath := range idx.Entries {
		if !workTreePaths[trackedPath] {
			absPath := filepath.Join(repo.Root, filepath.FromSlash(trackedPath))
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				idx.RemoveEntry(trackedPath)
			}
		}
	}

	return nil
}

func addDirectoryRecursive(repo *repository.Repository, idx *repository.Index, dirRelPath string) error {
	absDir := filepath.Join(repo.Root, filepath.FromSlash(dirRelPath))
	err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repo.Root, path)
		if err != nil {
			return err
		}
		norm := filesystem.NormalizePath(rel)
		if repo.Ignore.IsIgnored(norm, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			if err := stageSingleFile(repo, idx, norm, path, info); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func stageSingleFile(repo *repository.Repository, idx *repository.Index, normPath, absPath string, info os.FileInfo) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading file %s failed: %w", normPath, err)
	}

	blob := object.NewBlob(data)
	hash, err := repo.Objects.WriteObject(blob.Serialize())
	if err != nil {
		return fmt.Errorf("staging blob for %s failed: %w", normPath, err)
	}

	idx.AddOrUpdateEntry(repository.IndexEntry{
		Path:    normPath,
		Hash:    hash,
		Size:    info.Size(),
		Mode:    filesystem.GetFileMode(info),
		ModTime: info.ModTime(),
		Deleted: false,
	})

	return nil
}
