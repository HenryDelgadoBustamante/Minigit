package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"minigit/internal/filesystem"
	"minigit/internal/object"
	"minigit/internal/storage"
)

var ErrLocalChangesConflict = errors.New("your local changes to files would be overwritten by checkout")

type CheckoutResult struct {
	Target       string
	Branch       string
	CommitHash   string
	DetachedHEAD bool
	CommitMsg    string
}

// HasLocalChanges checks if there are unstaged modifications in the working tree.
func (r *Repository) HasLocalChanges() (bool, error) {
	idx, err := ReadIndex(r.Root)
	if err != nil {
		return false, err
	}

	headCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return false, err
	}

	var headTreeMap map[string]object.TreeEntry
	if headCommitHash != "" {
		commitObj, _, err := r.GetCommitByHash(headCommitHash)
		if err != nil {
			return false, err
		}
		headTreeMap, err = r.ReadTreeToMap(commitObj.Tree)
		if err != nil {
			return false, err
		}
	} else {
		headTreeMap = make(map[string]object.TreeEntry)
	}

	workItems, err := filesystem.WalkWorkingTree(r.Root, r.Ignore)
	if err != nil {
		return false, err
	}

	workTreeMap := make(map[string]filesystem.FileItem)
	for _, item := range workItems {
		workTreeMap[item.RelPath] = item
	}

	for _, idxEntry := range idx.SortedEntries() {
		if idxEntry.Deleted {
			continue
		}

		item, existsOnDisk := workTreeMap[idxEntry.Path]
		if !existsOnDisk {
			return true, nil
		}

		if item.Info.Size() != idxEntry.Size {
			return true, nil
		}

		data, err := os.ReadFile(item.AbsPath)
		if err == nil {
			blob := object.NewBlob(data)
			currentHash := storage.HashBytes(blob.Serialize())
			if currentHash != idxEntry.Hash {
				return true, nil
			}
		}
	}

	for headPath := range headTreeMap {
		if _, inIndex := idx.Entries[headPath]; !inIndex {
			return true, nil
		}
	}

	return false, nil
}

// Checkout switches to a target branch or commit hash.
func (r *Repository) Checkout(target string) (*CheckoutResult, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("no branch or commit hash specified")
	}

	lock, err := AcquireLock(GetIndexPath(r.Root))
	if err != nil {
		return nil, err
	}
	defer lock.Unlock()

	var targetBranch string
	var targetCommitHash string

	// Check if target is a branch name
	commitHashFromBranch, errBranch := ReadBranchCommit(r.Root, target)
	if errBranch == nil {
		targetBranch = target
		targetCommitHash = commitHashFromBranch
	} else {
		// Try to resolve as commit hash prefix
		resolvedHash, errHash := r.Objects.ResolveHash(target)
		if errHash != nil {
			if errors.Is(errBranch, ErrBranchNotFound) {
				return nil, fmt.Errorf("branch '%s' not found and '%s' is not a valid commit hash", target, target)
			}
			return nil, fmt.Errorf("pathspec or branch '%s' did not match any branch or valid commit object: %v", target, errHash)
		}
		targetCommitHash = resolvedHash
	}

	// Verify the target is a valid commit
	targetCommit, fullCommitHash, err := r.GetCommitByHash(targetCommitHash)
	if err != nil {
		return nil, fmt.Errorf("target '%s' is not a valid commit object: %w", target, err)
	}

	// Read target commit tree map
	targetTreeMap, err := r.ReadTreeToMap(targetCommit.Tree)
	if err != nil {
		return nil, fmt.Errorf("failed to read target commit tree: %w", err)
	}

	// Read current Index
	idx, err := ReadIndex(r.Root)
	if err != nil {
		return nil, err
	}

	// Read current HEAD commit tree map
	currentHeadCommitHash, _ := r.GetHeadCommitHash()
	var currentHeadTreeMap map[string]object.TreeEntry
	if currentHeadCommitHash != "" {
		if curCommit, _, err := r.GetCommitByHash(currentHeadCommitHash); err == nil {
			currentHeadTreeMap, _ = r.ReadTreeToMap(curCommit.Tree)
		}
	}
	if currentHeadTreeMap == nil {
		currentHeadTreeMap = make(map[string]object.TreeEntry)
	}

	// Check for local modifications that would be overwritten
	var conflictingFiles []string

	// Check files that exist in target commit
	for path, targetEntry := range targetTreeMap {
		idxEntry, inIndex := idx.Entries[path]
		curHeadEntry, inCurHead := currentHeadTreeMap[path]

		absPath := filepath.Join(r.Root, filepath.FromSlash(path))
		if info, err := os.Stat(absPath); err == nil {
			data, err := os.ReadFile(absPath)
			if err == nil {
				blob := object.NewBlob(data)
				currentDiskHash := storage.HashBytes(blob.Serialize())

				// Case 1: Untracked file on disk would be overwritten by target commit file
				if !inIndex && !inCurHead && currentDiskHash != targetEntry.Hash {
					conflictingFiles = append(conflictingFiles, path)
				} else if inIndex && idxEntry.Hash != targetEntry.Hash && currentDiskHash != targetEntry.Hash {
					// Case 2: Modified in working tree relative to target
					if inCurHead && curHeadEntry.Hash != currentDiskHash {
						conflictingFiles = append(conflictingFiles, path)
					}
				} else if !inIndex && inCurHead && curHeadEntry.Hash != currentDiskHash {
					// Case 3: File deleted from index but modified on disk
					conflictingFiles = append(conflictingFiles, path)
				}
			}

			_ = info
		}
	}

	// Check tracked files in current HEAD that are modified on disk
	for path, curHeadEntry := range currentHeadTreeMap {
		if _, inTarget := targetTreeMap[path]; inTarget {
			continue // Already checked above
		}

		absPath := filepath.Join(r.Root, filepath.FromSlash(path))
		if _, err := os.Stat(absPath); err == nil {
			data, err := os.ReadFile(absPath)
			if err == nil {
				blob := object.NewBlob(data)
				currentDiskHash := storage.HashBytes(blob.Serialize())
				if currentDiskHash != curHeadEntry.Hash {
					conflictingFiles = append(conflictingFiles, path)
				}
			}
		}
	}

	if len(conflictingFiles) > 0 {
		return nil, fmt.Errorf("%w:\n\t%s\nPlease commit your changes or restore them before switching", ErrLocalChangesConflict, strings.Join(conflictingFiles, "\n\t"))
	}

	// Perform checkout: restore target commit files
	for path, targetEntry := range targetTreeMap {
		raw, _, err := r.Objects.ReadObject(targetEntry.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to read blob for %s: %w", path, err)
		}

		blob, err := object.DecodeBlob(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to decode blob for %s: %w", path, err)
		}

		if err := filesystem.SafeWriteFile(r.Root, path, blob.Data, targetEntry.Mode); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	// Remove tracked files from current HEAD/Index that are missing in target commit
	filesToRemove := make(map[string]bool)
	for path := range currentHeadTreeMap {
		if _, inTarget := targetTreeMap[path]; !inTarget {
			filesToRemove[path] = true
		}
	}
	for path := range idx.Entries {
		if _, inTarget := targetTreeMap[path]; !inTarget {
			filesToRemove[path] = true
		}
	}
	for path := range filesToRemove {
		filesystem.SafeRemoveFile(r.Root, path)
	}

	// Rebuild index to match target commit tree entries
	newIdx := NewIndex()
	for path, entry := range targetTreeMap {
		absPath := filepath.Join(r.Root, filepath.FromSlash(path))
		var size int64
		var modTime time.Time = time.Now()
		if info, err := os.Stat(absPath); err == nil {
			size = info.Size()
			modTime = info.ModTime()
		}
		newIdx.AddOrUpdateEntry(IndexEntry{
			Path:    path,
			Hash:    entry.Hash,
			Size:    size,
			Mode:    entry.Mode,
			ModTime: modTime,
			Deleted: false,
		})
	}

	if err := WriteIndex(r.Root, newIdx); err != nil {
		return nil, fmt.Errorf("failed to update index during checkout: %w", err)
	}

	// Update HEAD
	firstLine := strings.Split(targetCommit.Message, "\n")[0]

	if targetBranch != "" {
		newHead := &HEAD{
			Type:   HEADTypeBranch,
			Branch: targetBranch,
		}
		if err := r.SetHEAD(newHead); err != nil {
			return nil, err
		}
		return &CheckoutResult{
			Target:       target,
			Branch:       targetBranch,
			CommitHash:   fullCommitHash,
			DetachedHEAD: false,
			CommitMsg:    firstLine,
		}, nil
	}

	newHead := &HEAD{
		Type:   HEADTypeDetached,
		Commit: fullCommitHash,
	}
	if err := r.SetHEAD(newHead); err != nil {
		return nil, err
	}

	return &CheckoutResult{
		Target:       target,
		Branch:       "",
		CommitHash:   fullCommitHash,
		DetachedHEAD: true,
		CommitMsg:    firstLine,
	}, nil
}
