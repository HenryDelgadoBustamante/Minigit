package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"minigit/internal/object"
	"minigit/internal/repository"
)

// RunShow shows detailed information for a specific commit hash.
func RunShow(repo *repository.Repository, hashPrefix string) error {
	if hashPrefix == "" {
		// Default to HEAD commit
		headHash, err := repo.GetHeadCommitHash()
		if err != nil || headHash == "" {
			return fmt.Errorf("no commit specified and HEAD has no commits yet")
		}
		hashPrefix = headHash
	}

	commitObj, fullHash, err := repo.GetCommitByHash(hashPrefix)
	if err != nil {
		return err
	}

	fmt.Printf("commit %s\n", fullHash)
	if commitObj.Parent != "" {
		fmt.Printf("parent %s\n", commitObj.Parent)
	}
	fmt.Printf("tree   %s\n", commitObj.Tree)
	fmt.Printf("Author: %s <%s>\n", commitObj.AuthorName, commitObj.AuthorMail)
	fmt.Printf("Date:   %s\n\n", commitObj.CreatedAt.Format(time.RFC1123Z))

	for _, line := range strings.Split(commitObj.Message, "\n") {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// Load tree map of current commit
	currentTreeMap, err := repo.ReadTreeToMap(commitObj.Tree)
	if err != nil {
		return fmt.Errorf("failed to read commit tree: %w", err)
	}

	// Compare with parent commit if available
	var parentTreeMap map[string]object.TreeEntry
	if commitObj.Parent != "" {
		parentCommit, _, err := repo.GetCommitByHash(commitObj.Parent)
		if err == nil {
			parentTreeMap, _ = repo.ReadTreeToMap(parentCommit.Tree)
		}
	}

	if parentTreeMap == nil {
		parentTreeMap = make(map[string]object.TreeEntry)
	}

	var added, modified, deleted []string

	for path, currentEntry := range currentTreeMap {
		parentEntry, inParent := parentTreeMap[path]
		if !inParent {
			added = append(added, path)
		} else if parentEntry.Hash != currentEntry.Hash {
			modified = append(modified, path)
		}
	}

	for path := range parentTreeMap {
		if _, inCurrent := currentTreeMap[path]; !inCurrent {
			deleted = append(deleted, path)
		}
	}

	sort.Strings(added)
	sort.Strings(modified)
	sort.Strings(deleted)

	if len(added) > 0 || len(modified) > 0 || len(deleted) > 0 {
		fmt.Println("Changes in this commit:")
		for _, p := range added {
			fmt.Printf("  + %s\n", p)
		}
		for _, p := range modified {
			fmt.Printf("  M %s\n", p)
		}
		for _, p := range deleted {
			fmt.Printf("  - %s\n", p)
		}
	}

	return nil
}
