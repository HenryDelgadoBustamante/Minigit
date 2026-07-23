package commands

import (
	"fmt"

	"minigit/internal/repository"
)

var ErrLocalChangesConflict = repository.ErrLocalChangesConflict

// RunCheckout switches to a target branch or commit hash.
func RunCheckout(repo *repository.Repository, target string) error {
	result, err := repo.Checkout(target)
	if err != nil {
		return err
	}

	if !result.DetachedHEAD {
		fmt.Printf("Switched to branch '%s'\n", result.Branch)
	} else {
		fmt.Printf("Note: switching to '%s'.\n", result.CommitHash[:7])
		fmt.Printf("You are in 'detached HEAD' state. HEAD is now at %s %s\n", result.CommitHash[:7], result.CommitMsg)
	}

	return nil
}
