package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"minigit/internal/filesystem"
	"minigit/internal/object"
	"minigit/internal/repository"
	"minigit/internal/storage"
)

var ErrLocalChangesConflict = errors.New("your local changes to files would be overwritten by checkout")

// RunCheckout switches to a target branch or commit hash.
func RunCheckout(repo *repository.Repository, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("no branch or commit hash specified")
	}

	lock, err := repository.AcquireLock(repository.GetIndexPath(repo.Root))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	var targetBranch string
	var targetCommitHash string

	// Check if target is a branch name
	commitHashFromBranch, errBranch := repository.ReadBranchCommit(repo.Root, target)
	if errBranch == nil {
		targetBranch = target
		targetCommitHash = commitHashFromBranch
	} else {
		// Resolve as commit hash prefix
		resolvedHash, errHash := repo.Objects.ResolveHash(target)
		if errHash != nil {
			return fmt.Errorf("pathspec or branch '%s' did not match any branch or valid commit object: %v", target, errHash)
		}
		targetCommitHash = resolvedHash
	}

	targetCommit, fullCommitHash, err := repo.GetCommitByHash(targetCommitHash)
	if err != nil {
		return fmt.Errorf("target '%s' is not a valid commit object: %w", target, err)
	}

	// Read target commit tree map
	targetTreeMap, err := repo.ReadTreeToMap(targetCommit.Tree)
	if err != nil {
		return fmt.Errorf("failed to read target commit tree: %w", err)
	}

	// Read current Index
	idx, err := repository.ReadIndex(repo.Root)
	if err != nil {
		return err
	}

	// Read current HEAD commit tree map
	currentHeadCommitHash, _ := repo.GetHeadCommitHash()
	var currentHeadTreeMap map[string]object.TreeEntry
	if currentHeadCommitHash != "" {
		if curCommit, _, err := repo.GetCommitByHash(currentHeadCommitHash); err == nil {
			currentHeadTreeMap, _ = repo.ReadTreeToMap(curCommit.Tree)
		}
	}
	if currentHeadTreeMap == nil {
		currentHeadTreeMap = make(map[string]object.TreeEntry)
	}

	// Check for local modifications that would be overwritten
	var conflictingFiles []string

	for path, targetEntry := range targetTreeMap {
		idxEntry, inIndex := idx.Entries[path]
		curHeadEntry, inCurHead := currentHeadTreeMap[path]

		// If file exists on disk
		absPath := filepath.Join(repo.Root, filepath.FromSlash(path))
		if info, err := os.Stat(absPath); err == nil {
			data, err := os.ReadFile(absPath)
			if err == nil {
				blob := object.NewBlob(data)
				currentDiskHash := storage.HashBytes(blob.Serialize())

				// Case 1: Untracked file on disk would be overwritten by target commit file
				if !inIndex && !inCurHead && currentDiskHash != targetEntry.Hash {
					conflictingFiles = append(conflictingFiles, path)
				} else if inIndex && idxEntry.Hash != targetEntry.Hash && currentDiskHash != targetEntry.Hash {
					// Case 2: Modified in working tree or index relative to target
					if inCurHead && curHeadEntry.Hash != currentDiskHash {
						conflictingFiles = append(conflictingFiles, path)
					}
				}
			}

			_ = info
		}
	}

	if len(conflictingFiles) > 0 {
		return fmt.Errorf("%w:\n\t%s\nPlease commit your changes or restore them before switching", ErrLocalChangesConflict, strings.Join(conflictingFiles, "\n\t"))
	}

	// Perform checkout: restore target commit files
	for path, targetEntry := range targetTreeMap {
		raw, _, err := repo.Objects.ReadObject(targetEntry.Hash)
		if err != nil {
			return fmt.Errorf("failed to read blob for %s: %w", path, err)
		}

		blob, err := object.DecodeBlob(raw)
		if err != nil {
			return fmt.Errorf("failed to decode blob for %s: %w", path, err)
		}

		if err := filesystem.SafeWriteFile(repo.Root, path, blob.Data, targetEntry.Mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	// Remove tracked files from current HEAD/Index that are missing in target commit
	for path := range currentHeadTreeMap {
		if _, inTarget := targetTreeMap[path]; !inTarget {
			filesystem.SafeRemoveFile(repo.Root, path)
		}
	}
	for path := range idx.Entries {
		if _, inTarget := targetTreeMap[path]; !inTarget {
			filesystem.SafeRemoveFile(repo.Root, path)
		}
	}

	// Rebuild index to match target commit tree entries
	newIdx := repository.NewIndex()
	for path, entry := range targetTreeMap {
		absPath := filepath.Join(repo.Root, filepath.FromSlash(path))
		var size int64
		var modTime time.Time = time.Now()
		if info, err := os.Stat(absPath); err == nil {
			size = info.Size()
			modTime = info.ModTime()
		}
		newIdx.AddOrUpdateEntry(repository.IndexEntry{
			Path:    path,
			Hash:    entry.Hash,
			Size:    size,
			Mode:    entry.Mode,
			ModTime: modTime,
			Deleted: false,
		})
	}

	if err := repository.WriteIndex(repo.Root, newIdx); err != nil {
		return fmt.Errorf("failed to update index during checkout: %w", err)
	}

	// Update HEAD
	if targetBranch != "" {
		newHead := &repository.HEAD{
			Type:   repository.HEADTypeBranch,
			Branch: targetBranch,
		}
		if err := repo.SetHEAD(newHead); err != nil {
			return err
		}
		fmt.Printf("Switched to branch '%s'\n", targetBranch)
	} else {
		newHead := &repository.HEAD{
			Type:   repository.HEADTypeDetached,
			Commit: fullCommitHash,
		}
		if err := repo.SetHEAD(newHead); err != nil {
			return err
		}
		fmt.Printf("Note: switching to '%s'.\n", fullCommitHash[:7])
		fmt.Printf("You are in 'detached HEAD' state. HEAD is now at %s %s\n", fullCommitHash[:7], strings.Split(targetCommit.Message, "\n")[0])
	}

	return nil
}
