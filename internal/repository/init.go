package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"minigit/internal/storage"
)

// InitRepository initializes a new MiniGit repository in targetDir and returns a Repository instance.
func InitRepository(targetDir string) (*Repository, error) {
	if targetDir == "" {
		targetDir = "."
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("invalid initialization directory: %w", err)
	}

	minigitDir := filepath.Join(absDir, ".minigit")
	if _, err := os.Stat(minigitDir); err == nil {
		return nil, fmt.Errorf("reinitialization error: repository already exists at %s", absDir)
	}

	// Create directories
	dirs := []string{
		minigitDir,
		filepath.Join(minigitDir, "objects"),
		filepath.Join(minigitDir, "refs", "heads"),
		filepath.Join(minigitDir, "logs"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create HEAD pointing to main
	head := &HEAD{
		Type:   HEADTypeBranch,
		Branch: "main",
	}
	if err := WriteHEAD(absDir, head); err != nil {
		return nil, fmt.Errorf("failed to write HEAD: %w", err)
	}

	// Create empty index
	emptyIdx := NewIndex()
	if err := WriteIndex(absDir, emptyIdx); err != nil {
		return nil, fmt.Errorf("failed to write initial index: %w", err)
	}

	// Create config file
	configPath := filepath.Join(minigitDir, "config")
	configContent := "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n"
	if err := storage.WriteFileAtomic(configPath, []byte(configContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return OpenRepository(absDir), nil
}
