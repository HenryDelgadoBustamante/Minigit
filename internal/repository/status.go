package repository

import (
	"os"
	"sort"

	"minigit/internal/filesystem"
	"minigit/internal/object"
	"minigit/internal/storage"
)

type FileStatusType string

const (
	StatusAdded    FileStatusType = "added"
	StatusModified FileStatusType = "modified"
	StatusDeleted  FileStatusType = "deleted"
)

type FileStatus struct {
	Path string
	Type FileStatusType
}

type StatusResult struct {
	Staged    []FileStatus
	Unstaged  []FileStatus
	Untracked []string
}

// GetStatus computes the status of the repository comparing Working Tree, Index, and HEAD.
func (r *Repository) GetStatus() (*StatusResult, error) {
	headCommitHash, err := r.GetHeadCommitHash()
	if err != nil {
		return nil, err
	}

	// 1. Load HEAD tree map
	var headTreeMap map[string]object.TreeEntry
	if headCommitHash != "" {
		commitObj, _, err := r.GetCommitByHash(headCommitHash)
		if err != nil {
			return nil, err
		}
		headTreeMap, err = r.ReadTreeToMap(commitObj.Tree)
		if err != nil {
			return nil, err
		}
	} else {
		headTreeMap = make(map[string]object.TreeEntry)
	}

	// 2. Load Index
	idx, err := ReadIndex(r.Root)
	if err != nil {
		return nil, err
	}

	result := &StatusResult{}

	// 3. Staged changes: Compare HEAD tree vs Index
	for _, idxEntry := range idx.SortedEntries() {
		headEntry, inHead := headTreeMap[idxEntry.Path]
		if idxEntry.Deleted {
			if inHead {
				result.Staged = append(result.Staged, FileStatus{Path: idxEntry.Path, Type: StatusDeleted})
			}
		} else {
			if !inHead {
				result.Staged = append(result.Staged, FileStatus{Path: idxEntry.Path, Type: StatusAdded})
			} else if headEntry.Hash != idxEntry.Hash {
				result.Staged = append(result.Staged, FileStatus{Path: idxEntry.Path, Type: StatusModified})
			}
		}
	}

	for headPath := range headTreeMap {
		if _, inIndex := idx.Entries[headPath]; !inIndex {
			result.Staged = append(result.Staged, FileStatus{Path: headPath, Type: StatusDeleted})
		}
	}

	// 4. Unstaged & Untracked changes: Compare Index vs Working Tree
	workItems, err := filesystem.WalkWorkingTree(r.Root, r.Ignore)
	if err != nil {
		return nil, err
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
			result.Unstaged = append(result.Unstaged, FileStatus{Path: idxEntry.Path, Type: StatusDeleted})
		} else {
			// Fast check size and modTime before reading full blob
			isModified := false
			if item.Info.Size() != idxEntry.Size {
				isModified = true
			} else {
				data, err := os.ReadFile(item.AbsPath)
				if err == nil {
					blob := object.NewBlob(data)
					currentHash := storage.HashBytes(blob.Serialize())
					if currentHash != idxEntry.Hash {
						isModified = true
					}
				}
			}
			if isModified {
				result.Unstaged = append(result.Unstaged, FileStatus{Path: idxEntry.Path, Type: StatusModified})
			}
		}
	}

	for relPath := range workTreeMap {
		if _, inIndex := idx.Entries[relPath]; !inIndex {
			result.Untracked = append(result.Untracked, relPath)
		}
	}

	// Sort results to be deterministic
	sort.Slice(result.Staged, func(i, j int) bool {
		return result.Staged[i].Path < result.Staged[j].Path
	})
	sort.Slice(result.Unstaged, func(i, j int) bool {
		return result.Unstaged[i].Path < result.Unstaged[j].Path
	})
	sort.Strings(result.Untracked)

	return result, nil
}
