package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrRepositoryNotFound = errors.New("not a minigit repository (or any of the parent directories): .minigit not found")

// DiscoverRepository walks upwards from startDir looking for a .minigit directory.
// Returns the absolute path of the repository root directory.
func DiscoverRepository(startDir string) (string, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Validate startDir does not contain null bytes
	if strings.Contains(absStart, "\x00") {
		return "", fmt.Errorf("invalid path: null bytes detected")
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
