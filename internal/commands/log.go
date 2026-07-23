package commands

import (
	"fmt"
	"strings"
	"time"

	"minigit/internal/repository"
)

// RunLog displays commit history starting from HEAD.
func RunLog(repo *repository.Repository, oneline bool) error {
	headCommitHash, err := repo.GetHeadCommitHash()
	if err != nil {
		return err
	}

	if headCommitHash == "" {
		return fmt.Errorf("your current branch does not have any commits yet")
	}

	head, err := repo.GetHEAD()
	if err != nil {
		return err
	}

	currHash := headCommitHash
	isFirst := true

	for currHash != "" {
		commitObj, fullHash, err := repo.GetCommitByHash(currHash)
		if err != nil {
			return fmt.Errorf("failed to read commit %s: %w", currHash, err)
		}

		shortHash := fullHash[:7]
		firstLine := strings.Split(commitObj.Message, "\n")[0]

		var refDecoration string
		if isFirst {
			if head.Type == repository.HEADTypeBranch {
				refDecoration = fmt.Sprintf(" (HEAD -> %s)", head.Branch)
			} else {
				refDecoration = " (HEAD)"
			}
			isFirst = false
		}

		if oneline {
			fmt.Printf("%s%s %s\n", shortHash, refDecoration, firstLine)
		} else {
			fmt.Printf("commit %s%s\n", fullHash, refDecoration)
			fmt.Printf("Author: %s <%s>\n", commitObj.AuthorName, commitObj.AuthorMail)
			fmt.Printf("Date:   %s\n\n", commitObj.CreatedAt.Format(time.RFC1123Z))
			for _, line := range strings.Split(commitObj.Message, "\n") {
				fmt.Printf("    %s\n", line)
			}
			fmt.Println()
		}

		currHash = commitObj.Parent
	}

	return nil
}
