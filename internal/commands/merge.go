package commands

import (
	"fmt"
	"strings"

	"minigit/internal/repository"
)

// RunMerge merges the target branch into the current branch using fast-forward.
func RunMerge(repo *repository.Repository, targetBranch string) error {
	targetBranch = strings.TrimSpace(targetBranch)
	if targetBranch == "" {
		return fmt.Errorf("error: se requiere el nombre de la rama a fusionar (ej: minigit merge <rama>)")
	}

	err := repo.MergeFastForward(targetBranch)
	if err != nil {
		return err
	}

	fmt.Println("Fast-forward realizado correctamente.")
	return nil
}
