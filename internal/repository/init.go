package repository

import (
	"fmt"
	"os"
	"path/filepath"

	"minigit/internal/filesystem"
	"minigit/internal/storage"
)

// InitRepository initializes a new MiniGit repository in targetDir.
// It returns the opened Repository instance on success.
func InitRepository(targetDir string) (*Repository, error) {
	// 1. Resolve target directory (empty means current directory)
	if targetDir == "" {
		targetDir = "."
	}
	// Normalize and get absolute path using existing filesystem helper
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("ruta inválida: %w", err)
	}
	// Clean path for consistency
	absDir = filesystem.NormalizePath(absDir)

	// 2. Detect existing repository
	minigitDir := filepath.Join(absDir, ".minigit")
	if _, err := os.Stat(minigitDir); err == nil {
		return nil, fmt.Errorf("Repositorio MiniGit ya inicializado en %s", absDir)
	}

	// 3. Verify we can write to the target directory
	if err := filesystem.IsWritable(absDir); err != nil {
		return nil, fmt.Errorf("permiso denegado para escribir en %s: %w", absDir, err)
	}

	// 4. Create required directories
	dirs := []string{
		minigitDir,
		filepath.Join(minigitDir, "objects"),
		filepath.Join(minigitDir, "refs"),
		filepath.Join(minigitDir, "refs", "heads"),
		filepath.Join(minigitDir, "logs"),
	}
	// Track creation for rollback
	created := []string{}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			cleanupRepository(created, minigitDir)
			return nil, fmt.Errorf("no fue posible crear el repositorio: error al crear %s: %w", d, err)
		}
		created = append(created, d)
	}

	// 5. Write HEAD (using existing helper without repository prefix)
	head := NewHEAD("main") // helper constructor if exists, otherwise manual
	if err := WriteHEAD(absDir, head); err != nil {
		cleanupRepository(created, minigitDir)
		return nil, fmt.Errorf("no fue posible crear el repositorio: error al escribir HEAD: %w", err)
	}

	// 6. Write empty index
	emptyIdx := NewIndex()
	if err := WriteIndex(absDir, emptyIdx); err != nil {
		cleanupRepository(created, minigitDir)
		return nil, fmt.Errorf("no fue posible crear el repositorio: error al escribir index: %w", err)
	}

	// 7. Write config file (reuse any existing helper, otherwise inline)
	configPath := filepath.Join(minigitDir, "config")
	configContent := "[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n\tbare = false\n"
	if err := storage.WriteFileAtomic(configPath, []byte(configContent), 0644); err != nil {
		cleanupRepository(created, minigitDir)
		return nil, fmt.Errorf("no fue posible crear el repositorio: error al escribir config: %w", err)
	}

	// 8. Return opened repository
	return OpenRepository(absDir), nil
}

// cleanupRepository removes only the resources created during the current init execution.
func cleanupRepository(created []string, root string) {
	// If the root .minigit directory was created, remove it entirely.
	// This also removes any sub‑directories we may have added.
	_ = os.RemoveAll(root)
	// Individual removal of created dirs is unnecessary because removing root cleans them up.
}
