package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"minigit/internal/repository"
	"minigit/internal/storage"
)

// RunInit initializes a new MiniGit repository in targetDir.
func RunInit(targetDir string) error {
	if targetDir == "" {
		targetDir = "."
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("invalid initialization directory: %w", err)
	}

	minigitDir := filepath.Join(absDir, ".minigit")
	if _, err := os.Stat(minigitDir); err == nil {
		return fmt.Errorf("reinitialization error: repository already exists at %s", absDir)
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
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create HEAD pointing to main
	head := &repository.HEAD{
		Type:   repository.HEADTypeBranch,
		Branch: "main",
	}
	if err := repository.WriteHEAD(absDir, head); err != nil {
		return fmt.Errorf("failed to write HEAD: %w", err)
	}

	// Create initial main ref directory if needed, or leave ref unwritten until initial commit
	// Create empty index
	emptyIdx := repository.NewIndex()
	if err := repository.WriteIndex(absDir, emptyIdx); err != nil {
		return fmt.Errorf("failed to write initial index: %w", err)
	}

	// Create config file
	configPath := filepath.Join(minigitDir, "config")
	configContent := "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n"
	if err := storage.WriteFileAtomic(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Initialized empty MiniGit repository in %s\n", minigitDir)
	return nil
}
