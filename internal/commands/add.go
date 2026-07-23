package commands

import (
	"minigit/internal/repository"
)

// RunAdd stages specified files or directories.
func RunAdd(repo *repository.Repository, paths []string) error {
	return repo.Add(paths)
}
