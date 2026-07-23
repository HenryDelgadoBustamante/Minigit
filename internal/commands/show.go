package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"minigit/internal/object"
	"minigit/internal/repository"
)

// RunShow shows detailed information for a specific object hash (Blob, Tree, or Commit).
func RunShow(repo *repository.Repository, hashPrefix string) error {
	res, err := repo.InspectObject(hashPrefix)
	if err != nil {
		return err
	}

	switch res.Type {
	case object.TypeBlob:
		_, err := os.Stdout.Write(res.BlobData)
		return err

	case object.TypeTree:
		for _, entry := range res.Tree.Entries {
			fmt.Printf("%06o %s %s\t%s\n", entry.Mode, entry.Type, entry.Hash, entry.Name)
		}
		return nil

	case object.TypeCommit:
		commitObj := res.Commit
		fmt.Printf("commit %s\n", res.FullHash)
		fmt.Printf("tree   %s\n", commitObj.Tree)
		if commitObj.Parent != "" {
			fmt.Printf("parent %s\n", commitObj.Parent)
		}
		fmt.Printf("Author: %s <%s>\n", commitObj.AuthorName, commitObj.AuthorMail)
		fmt.Printf("Date:   %s\n\n", commitObj.CreatedAt.Format(time.RFC1123Z))

		for _, line := range strings.Split(commitObj.Message, "\n") {
			fmt.Printf("    %s\n", line)
		}
		fmt.Println()

		if res.CommitDiff != nil {
			diff := res.CommitDiff
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
		}
		return nil

	default:
		return fmt.Errorf("tipo de objeto no soportado: %s", res.Type)
	}
}
