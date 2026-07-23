package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiffAndRenameDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "minigit-diff-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := InitRepository(tempDir)
	if err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	// Commit 1: Add archivo.txt and antiguo.txt
	file1Path := filepath.Join(tempDir, "archivo.txt")
	fileOldPath := filepath.Join(tempDir, "antiguo.txt")
	if err := os.WriteFile(file1Path, []byte("Línea 1\nLínea 2\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(fileOldPath, []byte("Contenido para renombrar\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := repo.Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	res1, err := repo.Commit("Commit 1: estado inicial", "Autor", "autor@dev.org")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Commit 2: Modify archivo.txt, delete antiguo.txt, add renombrado.txt with same content as antiguo.txt (Rename!), add totalmente_nuevo.txt
	if err := os.WriteFile(file1Path, []byte("Línea 1\nLínea 2 modificada\nLínea 3 agregada\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	os.Remove(fileOldPath)

	fileRenamedPath := filepath.Join(tempDir, "renombrado.txt")
	if err := os.WriteFile(fileRenamedPath, []byte("Contenido para renombrar\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	fileNewPath := filepath.Join(tempDir, "totalmente_nuevo.txt")
	if err := os.WriteFile(fileNewPath, []byte("Nuevo archivo\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := repo.Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	res2, err := repo.Commit("Commit 2: cambios y renombrado", "Autor", "autor@dev.org")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Run CompareCommits
	res, err := repo.CompareCommits(res1.Hash, res2.Hash)
	if err != nil {
		t.Fatalf("CompareCommits failed: %v", err)
	}

	changesMap := make(map[string]ChangeType)
	for _, c := range res.Changes {
		changesMap[c.Path] = c.Type
		if c.Type == ChangeRenamed {
			if c.OldPath != "antiguo.txt" || c.Path != "renombrado.txt" {
				t.Errorf("expected rename antiguo.txt -> renombrado.txt, got %s -> %s", c.OldPath, c.Path)
			}
		}
	}

	if changesMap["archivo.txt"] != ChangeModified {
		t.Errorf("expected archivo.txt to be Modified, got %s", changesMap["archivo.txt"])
	}
	if changesMap["renombrado.txt"] != ChangeRenamed {
		t.Errorf("expected renombrado.txt to be Renamed, got %s", changesMap["renombrado.txt"])
	}
	if changesMap["totalmente_nuevo.txt"] != ChangeAdded {
		t.Errorf("expected totalmente_nuevo.txt to be Added, got %s", changesMap["totalmente_nuevo.txt"])
	}
}

func TestLineDiff(t *testing.T) {
	oldContent := []byte("Linea 1\nLinea 2\nLinea 3\n")
	newContent := []byte("Linea 1\nLinea 2 modificada\nLinea 3\nLinea 4 nueva\n")

	diffLines, isBinary := GetFileDiffLines(oldContent, newContent)
	if isBinary {
		t.Fatalf("expected text diff, got binary")
	}

	if len(diffLines) == 0 {
		t.Fatalf("expected non-empty diff lines")
	}
}
