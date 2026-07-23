package commands

import (
	"errors"
	"fmt"
	"os"

	"minigit/internal/repository"
)

var (
	ErrEmptyCommitMessage = errors.New("commit message cannot be empty")
	ErrNothingToCommit    = errors.New("nothing to commit, working tree clean")
)

// RunCommit creates a new commit from the staged index.
func RunCommit(repo *repository.Repository, message string) error {
	authorName := os.Getenv("MINIGIT_AUTHOR_NAME")
	authorEmail := os.Getenv("MINIGIT_AUTHOR_EMAIL")

	result, err := repo.Commit(message, authorName, authorEmail)
	if err != nil {
		if errors.Is(err, repository.ErrEmptyCommitMessage) {
			return ErrEmptyCommitMessage
		}
		if errors.Is(err, repository.ErrNothingToCommit) {
			return ErrNothingToCommit
		}
		return err
	}

	if result.Branch != "" {
		fmt.Printf("[%s %s] %s\n", result.Branch, result.ShortHash, result.Message)
	} else {
		fmt.Printf("[detached HEAD %s] %s\n", result.ShortHash, result.Message)
	}

	return nil
}
