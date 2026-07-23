package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrRepositoryNotFound = errors.New("not a minigit repository (or any of the parent directories): .minigit not found")

// DiscoverRepository walks upwards from startDir looking for a .minigit directory.
// Returns the absolute path of the repository root directory.
func DiscoverRepository(startDir string) (string, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	curr := absStart
	for {
		minigitDir := filepath.Join(curr, ".minigit")
		info, err := os.Stat(minigitDir)
		if err == nil && info.IsDir() {
			return curr, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return "", ErrRepositoryNotFound
}
