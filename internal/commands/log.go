package commands

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"minigit/internal/repository"
)

// RunLog displays commit history starting from HEAD.
func RunLog(repo *repository.Repository, oneline bool) error {
	entries, err := repo.GetCommitHistory()
	if err != nil {
		if errors.Is(err, repository.ErrNoCommits) {
			fmt.Println("no hay commits registrados en este repositorio")
			return nil
		}
		return err
	}

	head, err := repo.GetHEAD()
	if err != nil {
		return err
	}

	for i, entry := range entries {
		shortHash := entry.Hash[:7]
		firstLine := strings.Split(entry.Message, "\n")[0]

		var refDecoration string
		if i == 0 {
			if head.Type == repository.HEADTypeBranch {
				refDecoration = fmt.Sprintf(" (HEAD -> %s)", head.Branch)
			} else {
				refDecoration = " (HEAD)"
			}
		}

		if oneline {
			fmt.Printf("%s%s %s\n", shortHash, refDecoration, firstLine)
		} else {
			fmt.Printf("commit %s%s\n", entry.Hash, refDecoration)
			fmt.Printf("Author: %s <%s>\n", entry.AuthorName, entry.AuthorEmail)
			fmt.Printf("Date:   %s\n\n", entry.Timestamp.Format(time.RFC1123Z))
			for _, line := range strings.Split(entry.Message, "\n") {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}
	}

	return nil
}
