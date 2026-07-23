package commands

import (
	"fmt"
	"strings"
	"time"

	"minigit/internal/repository"
)

// RunShow shows detailed information for a specific commit hash.
func RunShow(repo *repository.Repository, hashPrefix string) error {
	diff, commitObj, fullHash, err := repo.GetCommitDiff(hashPrefix)
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

	if len(diff.Added) > 0 || len(diff.Modified) > 0 || len(diff.Deleted) > 0 {
		fmt.Println("Changes in this commit:")
		for _, p := range diff.Added {
			fmt.Printf("  + %s\n", p)
		}
		for _, p := range diff.Modified {
			fmt.Printf("  M %s\n", p)
		}
		for _, p := range diff.Deleted {
			fmt.Printf("  - %s\n", p)
		}
	}

	return nil
}
