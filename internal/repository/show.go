package repository

import (
	"fmt"
	"sort"
	"strings"

	"minigit/internal/object"
)

type CommitDiff struct {
	Added    []string
	Modified []string
	Deleted  []string
}

// GetCommitDiff retrieves a commit by prefix, resolving default hash, and computes diff against its parent commit.
func (r *Repository) GetCommitDiff(hashPrefix string) (*CommitDiff, *object.Commit, string, error) {
	if hashPrefix == "" || strings.EqualFold(hashPrefix, "HEAD") {
		// Default to HEAD commit
		headHash, err := r.GetHeadCommitHash()
		if err != nil {
			return nil, nil, "", err
		}
		if headHash == "" {
			return nil, nil, "", fmt.Errorf("no commit specified and HEAD has no commits yet")
		}
		hashPrefix = headHash
	}

	commitObj, fullHash, err := r.GetCommitByHash(hashPrefix)
	if err != nil {
		return nil, nil, "", err
	}

	// Load tree map of current commit
	currentTreeMap, err := r.ReadTreeToMap(commitObj.Tree)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to read commit tree: %w", err)
	}

	// Compare with parent commit if available
	var parentTreeMap map[string]object.TreeEntry
	if commitObj.Parent != "" {
		parentCommit, _, err := r.GetCommitByHash(commitObj.Parent)
		if err == nil {
			parentTreeMap, _ = r.ReadTreeToMap(parentCommit.Tree)
		}
	}

	if parentTreeMap == nil {
		parentTreeMap = make(map[string]object.TreeEntry)
	}

	diff := &CommitDiff{}

	for path, currentEntry := range currentTreeMap {
		parentEntry, inParent := parentTreeMap[path]
		if !inParent {
			diff.Added = append(diff.Added, path)
		} else if parentEntry.Hash != currentEntry.Hash {
			diff.Modified = append(diff.Modified, path)
		}
	}

	for path := range parentTreeMap {
		if _, inCurrent := currentTreeMap[path]; !inCurrent {
			diff.Deleted = append(diff.Deleted, path)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Modified)
	sort.Strings(diff.Deleted)

	return diff, commitObj, fullHash, nil
}
