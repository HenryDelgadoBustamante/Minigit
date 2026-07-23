package commands

import (
	"minigit/internal/repository"
)

// RunRestore restores a file in working directory or staged index from HEAD.
func RunRestore(repo *repository.Repository, targetPath string, staged bool) error {
	return repo.Restore(targetPath, staged)
}
