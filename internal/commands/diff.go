package commands

import (
	"fmt"

	"minigit/internal/repository"
)

// RunDiff compares two commits and outputs structural and line-by-line differences.
func RunDiff(repo *repository.Repository, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: minigit diff <commit-1> [commit-2]")
	}

	commit1 := args[0]
	commit2 := "HEAD"
	if len(args) >= 2 {
		commit2 = args[1]
	}

	res, err := repo.CompareCommits(commit1, commit2)
	if err != nil {
		return err
	}

	if len(res.Changes) == 0 {
		return nil
	}

	// 1. Output summary table (A, M, D, R)
	for _, change := range res.Changes {
		switch change.Type {
		case repository.ChangeAdded:
			fmt.Printf("A  %s\n", change.Path)
		case repository.ChangeModified:
			fmt.Printf("M  %s\n", change.Path)
		case repository.ChangeDeleted:
			fmt.Printf("D  %s\n", change.Path)
		case repository.ChangeRenamed:
			fmt.Printf("R  %s -> %s\n", change.OldPath, change.Path)
		}
	}

	// 2. Output detailed line diffs for text files
	fmt.Println()
	for _, change := range res.Changes {
		oldData, _ := repo.GetBlobContent(change.OldHash)
		newData, _ := repo.GetBlobContent(change.NewHash)

		diffLines, isBinary := repository.GetFileDiffLines(oldData, newData)
		if isBinary {
			fmt.Printf("--- a/%s\n+++ b/%s\nArchivos binarios difieren\n\n", change.Path, change.Path)
			continue
		}

		if len(diffLines) > 0 {
			fmt.Printf("--- a/%s\n", change.Path)
			fmt.Printf("+++ b/%s\n", change.Path)
			for _, line := range diffLines {
				fmt.Println(line)
			}
			fmt.Println()
		}
	}

	return nil
}
