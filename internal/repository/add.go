package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"minigit/internal/filesystem"
	"minigit/internal/object"
)

// Add stages specified files or directories to the index.
func (r *Repository) Add(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("nothing specified, nothing added")
	}

	lock, err := AcquireLock(GetIndexPath(r.Root))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	idx, err := ReadIndex(r.Root)
	if err != nil {
		return err
	}

	for _, p := range paths {
		if err := r.addPath(idx, p); err != nil {
			return err
		}
	}

	return WriteIndex(r.Root, idx)
}

func (r *Repository) addPath(idx *Index, argPath string) error {
	if argPath == "." {
		return r.addAllRecursive(idx)
	}

	normPath, err := filesystem.ValidateRelativePath(argPath, r.Root)
	if err != nil {
		return fmt.Errorf("invalid path '%s': %w", argPath, err)
	}

	absPath := filepath.Join(r.Root, filepath.FromSlash(normPath))
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

	if r.Ignore.IsIgnored(normPath, info.IsDir()) {
		return nil
	}

	if info.IsDir() {
		return r.addDirectoryRecursive(idx, normPath)
	}

	return r.stageSingleFile(idx, normPath, absPath, info)
}

func (r *Repository) addAllRecursive(idx *Index) error {
	items, err := filesystem.WalkWorkingTree(r.Root, r.Ignore)
	if err != nil {
		return fmt.Errorf("failed to walk repository tree: %w", err)
	}

	workTreePaths := make(map[string]bool)
	for _, item := range items {
		workTreePaths[item.RelPath] = true
		if err := r.stageSingleFile(idx, item.RelPath, item.AbsPath, item.Info); err != nil {
			return err
		}
	}

	// Detect deleted tracked files in working directory
	for trackedPath := range idx.Entries {
		if !workTreePaths[trackedPath] {
			absPath := filepath.Join(r.Root, filepath.FromSlash(trackedPath))
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				idx.RemoveEntry(trackedPath)
			}
		}
	}

	return nil
}

func (r *Repository) addDirectoryRecursive(idx *Index, dirRelPath string) error {
	absDir := filepath.Join(r.Root, filepath.FromSlash(dirRelPath))
	err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(r.Root, path)
		if err != nil {
			return err
		}
		norm := filesystem.NormalizePath(rel)
		if r.Ignore.IsIgnored(norm, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			if err := r.stageSingleFile(idx, norm, path, info); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (r *Repository) stageSingleFile(idx *Index, normPath, absPath string, info os.FileInfo) error {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading file %s failed: %w", normPath, err)
	}

	blob := object.NewBlob(data)
	hash, err := r.Objects.WriteObject(blob.Serialize())
	if err != nil {
		return fmt.Errorf("staging blob for %s failed: %w", normPath, err)
	}

	idx.AddOrUpdateEntry(IndexEntry{
		Path:    normPath,
		Hash:    hash,
		Size:    info.Size(),
		Mode:    filesystem.GetFileMode(info),
		ModTime: info.ModTime(),
		Deleted: false,
	})

	return nil
}
