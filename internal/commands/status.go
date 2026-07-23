package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"minigit/internal/filesystem"
	"minigit/internal/object"
	"minigit/internal/repository"
	"minigit/internal/storage"
)

type StatusResult struct {
	BranchName     string
	IsDetached     bool
	HeadHash       string
	StagedAdded    []string
	StagedModified []string
	StagedDeleted  []string
	UnstagedMod    []string
	UnstagedDel    []string
	Untracked      []string
}

// RunStatus returns the formatted status output.
func RunStatus(repo *repository.Repository) (string, error) {
	status, err := GetStatus(repo)
	if err != nil {
		return "", err
	}
	return FormatStatus(status), nil
}

func GetStatus(repo *repository.Repository) (*StatusResult, error) {
	head, err := repo.GetHEAD()
	if err != nil {
		return nil, err
	}

	result := &StatusResult{}

	if head.Type == repository.HEADTypeBranch {
		result.BranchName = head.Branch
	} else {
		result.IsDetached = true
		result.HeadHash = head.Commit[:7]
	}

	headCommitHash, err := repo.GetHeadCommitHash()
	if err != nil {
		return nil, err
	}

	// 1. Load HEAD tree map
	var headTreeMap map[string]object.TreeEntry
	if headCommitHash != "" {
		commitObj, _, err := repo.GetCommitByHash(headCommitHash)
		if err != nil {
			return nil, err
		}
		headTreeMap, err = repo.ReadTreeToMap(commitObj.Tree)
		if err != nil {
			return nil, err
		}
	} else {
		headTreeMap = make(map[string]object.TreeEntry)
	}

	// 2. Load Index
	idx, err := repository.ReadIndex(repo.Root)
	if err != nil {
		return nil, err
	}

	// 3. Staged changes: Compare HEAD tree vs Index
	for _, idxEntry := range idx.SortedEntries() {
		headEntry, inHead := headTreeMap[idxEntry.Path]
		if idxEntry.Deleted {
			if inHead {
				result.StagedDeleted = append(result.StagedDeleted, idxEntry.Path)
			}
		} else {
			if !inHead {
				result.StagedAdded = append(result.StagedAdded, idxEntry.Path)
			} else if headEntry.Hash != idxEntry.Hash {
				result.StagedModified = append(result.StagedModified, idxEntry.Path)
			}
		}
	}

	for headPath := range headTreeMap {
		if _, inIndex := idx.Entries[headPath]; !inIndex {
			result.StagedDeleted = append(result.StagedDeleted, headPath)
		}
	}

	// 4. Unstaged & Untracked changes: Compare Index vs Working Tree
	workItems, err := filesystem.WalkWorkingTree(repo.Root, repo.Ignore)
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
			result.UnstagedDel = append(result.UnstagedDel, idxEntry.Path)
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
				result.UnstagedMod = append(result.UnstagedMod, idxEntry.Path)
			}
		}
	}

	for relPath := range workTreeMap {
		if _, inIndex := idx.Entries[relPath]; !inIndex {
			result.Untracked = append(result.Untracked, relPath)
		}
	}

	sort.Strings(result.StagedAdded)
	sort.Strings(result.StagedModified)
	sort.Strings(result.StagedDeleted)
	sort.Strings(result.UnstagedMod)
	sort.Strings(result.UnstagedDel)
	sort.Strings(result.Untracked)

	return result, nil
}

func FormatStatus(s *StatusResult) string {
	var sb strings.Builder

	if s.IsDetached {
		sb.WriteString(fmt.Sprintf("HEAD detached at %s\n", s.HeadHash))
	} else {
		sb.WriteString(fmt.Sprintf("On branch %s\n", s.BranchName))
	}

	hasStaged := len(s.StagedAdded) > 0 || len(s.StagedModified) > 0 || len(s.StagedDeleted) > 0
	hasUnstaged := len(s.UnstagedMod) > 0 || len(s.UnstagedDel) > 0
	hasUntracked := len(s.Untracked) > 0

	if hasStaged {
		sb.WriteString("\nChanges to be committed:\n")
		for _, p := range s.StagedAdded {
			sb.WriteString(fmt.Sprintf("  new file:   %s\n", p))
		}
		for _, p := range s.StagedModified {
			sb.WriteString(fmt.Sprintf("  modified:   %s\n", p))
		}
		for _, p := range s.StagedDeleted {
			sb.WriteString(fmt.Sprintf("  deleted:    %s\n", p))
		}
	}

	if hasUnstaged {
		sb.WriteString("\nChanges not staged for commit:\n")
		for _, p := range s.UnstagedMod {
			sb.WriteString(fmt.Sprintf("  modified:   %s\n", p))
		}
		for _, p := range s.UnstagedDel {
			sb.WriteString(fmt.Sprintf("  deleted:    %s\n", p))
		}
	}

	if hasUntracked {
		sb.WriteString("\nUntracked files:\n")
		for _, p := range s.Untracked {
			sb.WriteString(fmt.Sprintf("  %s\n", p))
		}
	}

	if !hasStaged && !hasUnstaged && !hasUntracked {
		sb.WriteString("nothing to commit, working tree clean\n")
	}

	return sb.String()
}
