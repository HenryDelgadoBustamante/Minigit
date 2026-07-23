package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"minigit/internal/filesystem"
	"minigit/internal/object"
)

// Restore restores a file in the working directory or staged index from HEAD.
func (r *Repository) Restore(targetPath string, staged bool) error {
	normPath, err := filesystem.ValidateRelativePath(targetPath, r.Root)
	if err != nil {
		return fmt.Errorf("invalid path for restore: %w", err)
	}

	headCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return err
	}

	var headTreeMap map[string]object.TreeEntry
	if headCommitHash != "" {
		commitObj, _, err := r.GetCommitByHash(headCommitHash)
		if err != nil {
			return err
		}
		headTreeMap, err = r.ReadTreeToMap(commitObj.Tree)
		if err != nil {
			return err
		}
	} else {
		headTreeMap = make(map[string]object.TreeEntry)
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

	headEntry, existsInHEAD := headTreeMap[normPath]

	if staged {
		// Restore index entry from HEAD commit tree
		if !existsInHEAD {
			idx.RemoveEntry(normPath)
		} else {
			idx.AddOrUpdateEntry(IndexEntry{
				Path:    normPath,
				Hash:    headEntry.Hash,
				Mode:    headEntry.Mode,
				ModTime: time.Now(),
				Deleted: false,
			})
		}
		return WriteIndex(r.Root, idx)
	}

	// Restore working directory file from index (or HEAD if not in index)
	var targetBlobHash string
	var targetMode uint32 = 0644

	if idxEntry, inIndex := idx.Entries[normPath]; inIndex && !idxEntry.Deleted {
		targetBlobHash = idxEntry.Hash
		targetMode = idxEntry.Mode
	} else if existsInHEAD {
		targetBlobHash = headEntry.Hash
		targetMode = headEntry.Mode
	} else {
		return fmt.Errorf("pathspec '%s' did not match any file in index or HEAD", targetPath)
	}

	raw, _, err := r.Objects.ReadObject(targetBlobHash)
	if err != nil {
		return fmt.Errorf("failed to read blob %s: %w", targetBlobHash, err)
	}

	blob, err := object.DecodeBlob(raw)
	if err != nil {
		return fmt.Errorf("failed to decode blob %s: %w", targetBlobHash, err)
	}

	if err := filesystem.SafeWriteFile(r.Root, normPath, blob.Data, targetMode); err != nil {
		return fmt.Errorf("failed to restore file on disk: %w", err)
	}

	// Update index modTime and size for restored file
	absPath := filepath.Join(r.Root, filepath.FromSlash(normPath))
	if info, err := os.Stat(absPath); err == nil {
		idx.AddOrUpdateEntry(IndexEntry{
			Path:    normPath,
			Hash:    targetBlobHash,
			Size:    info.Size(),
			Mode:    filesystem.GetFileMode(info),
			ModTime: info.ModTime(),
			Deleted: false,
		})
		_ = WriteIndex(r.Root, idx)
	}

	return nil
}
