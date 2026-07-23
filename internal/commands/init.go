package commands

import (
	"fmt"
	"path/filepath"

	"minigit/internal/repository"
)

// RunInit initializes a new MiniGit repository in targetDir.
func RunInit(targetDir string) error {
	repo, err := repository.InitRepository(targetDir)
	if err != nil {
		return err
	}

	minigitDir := filepath.Join(repo.Root, ".minigit")
	fmt.Printf("Initialized empty MiniGit repository in %s\n", minigitDir)
	return nil
}
