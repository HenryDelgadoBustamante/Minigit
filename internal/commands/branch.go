package commands

import (
	"fmt"
	"strings"

	"minigit/internal/repository"
)

// RunBranch lists or creates branches.
func RunBranch(repo *repository.Repository, branchName string) error {
	branchName = strings.TrimSpace(branchName)

	if branchName == "" {
		// List branches
		branches, err := repo.ListBranches()
		if err != nil {
			return err
		}

		head, err := repo.GetHEAD()
		if err != nil {
			return err
		}

		if len(branches) == 0 {
			fmt.Println("No branches found.")
			return nil
		}

		for _, b := range branches {
			commitHash, err := repository.ReadBranchCommit(repo.Root, b)
			shortHash := "???????"
			if err == nil && len(commitHash) >= 7 {
				shortHash = commitHash[:7]
			}

			if head.Type == repository.HEADTypeBranch && b == head.Branch {
				fmt.Printf("* %s (%s)\n", b, shortHash)
			} else {
				fmt.Printf("  %s (%s)\n", b, shortHash)
			}
		}

		return nil
	}

	// Create branch
	if err := repository.ValidateBranchName(branchName); err != nil {
		return err
	}

	headCommitHash, err := repo.GetHeadCommitHash()
	if err != nil || headCommitHash == "" {
		return fmt.Errorf("cannot create branch '%s': current branch has no commits yet", branchName)
	}

	if err := repo.CreateBranch(branchName, headCommitHash); err != nil {
		return err
	}

	fmt.Printf("Created branch '%s' at %s\n", branchName, headCommitHash[:7])
	return nil
}
