package repository_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"minigit/internal/object"
	"minigit/internal/repository"
)

func TestIgnoreMatcher(t *testing.T) {
	repoRoot := t.TempDir()

	ignoreContent := `
# Comments
*.log
temp/
!keep.log
config.json
`
	os.WriteFile(filepath.Join(repoRoot, ".minigitignore"), []byte(ignoreContent), 0644)

	matcher := repository.NewIgnoreMatcher(repoRoot)

	// Test always ignored internal folders
	if !matcher.IsIgnored(".minigit/HEAD", false) {
		t.Fatalf(".minigit should be ignored")
	}

	// Test *.log extension ignore rule
	if !matcher.IsIgnored("app.log", false) {
		t.Fatalf("app.log should be ignored")
	}

	// Test negation rule !keep.log
	if matcher.IsIgnored("keep.log", false) {
		t.Fatalf("keep.log should NOT be ignored due to ! rule")
	}

	// Test directory rule temp/
	if !matcher.IsIgnored("temp", true) {
		t.Fatalf("temp directory should be ignored")
	}

	// Test exact file match config.json
	if !matcher.IsIgnored("config.json", false) {
		t.Fatalf("config.json should be ignored")
	}

	// Test unignored file
	if matcher.IsIgnored("main.go", false) {
		t.Fatalf("main.go should NOT be ignored")
	}
}

func TestSimultaneousLock(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "index")

	lock1, err := repository.AcquireLock(targetFile)
	if err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}
	defer lock1.Unlock()

	// Try acquiring second lock simultaneously
	_, err = repository.AcquireLock(targetFile)
	if !errors.Is(err, repository.ErrLockExists) {
		t.Fatalf("expected ErrLockExists, got %v", err)
	}
}

func TestCorruptIndexDetection(t *testing.T) {
	repoRoot := t.TempDir()
	minigitDir := filepath.Join(repoRoot, ".minigit")
	os.MkdirAll(minigitDir, 0755)

	indexPath := filepath.Join(minigitDir, "index")
	os.WriteFile(indexPath, []byte("{invalid json index data"), 0644)

	_, err := repository.ReadIndex(repoRoot)
	if !errors.Is(err, repository.ErrCorruptIndex) {
		t.Fatalf("expected ErrCorruptIndex, got %v", err)
	}
}

func TestBranchNameValidation(t *testing.T) {
	invalidNames := []string{"", "..", "feature..test", "/leading", "trailing/", "with space", "bad:char"}
	for _, name := range invalidNames {
		if err := repository.ValidateBranchName(name); err == nil {
			t.Fatalf("expected error for invalid branch name '%s'", name)
		}
	}

	validNames := []string{"main", "feature/my-branch", "v1.0.0", "fix-bug"}
	for _, name := range validNames {
		if err := repository.ValidateBranchName(name); err != nil {
			t.Fatalf("expected valid branch name '%s', got error: %v", name, err)
		}
	}
}

func TestMerkleGraphPropagationAndImmutability(t *testing.T) {
	// 1. Inicializar repositorio
	repoDir := t.TempDir()
	if _, err := repository.InitRepository(repoDir); err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}
	repo := repository.OpenRepository(repoDir)

	// 2. Crear los archivos
	readmePath := filepath.Join(repoDir, "README.md")
	mainPath := filepath.Join(repoDir, "src", "main.go")
	hashPath := filepath.Join(repoDir, "src", "utils", "hash.go")
	docsPath := filepath.Join(repoDir, "docs", "manual.md")

	os.MkdirAll(filepath.Dir(mainPath), 0755)
	os.MkdirAll(filepath.Dir(hashPath), 0755)
	os.MkdirAll(filepath.Dir(docsPath), 0755)

	os.WriteFile(readmePath, []byte("# Proyecto Merkle"), 0644)
	os.WriteFile(mainPath, []byte("package main\nfunc main() {}"), 0644)
	os.WriteFile(hashPath, []byte("package utils\nfunc Hash() {}"), 0644)
	os.WriteFile(docsPath, []byte("# Manual"), 0644)

	// 3. Ejecutar add
	if err := repo.Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// 4. Crear commit inicial
	res1, err := repo.Commit("Commit inicial", "Tester", "test@minigit.local")
	if err != nil {
		t.Fatalf("Commit 1 failed: %v", err)
	}

	// 5. Leer el commit almacenado
	commit1, _, err := repo.GetCommitByHash(res1.Hash)
	if err != nil {
		t.Fatalf("GetCommitByHash 1 failed: %v", err)
	}

	// 6. Obtener tree raíz y recorrer subtrees
	treeMap1, err := repo.ReadTreeToMap(commit1.Tree)
	if err != nil {
		t.Fatalf("ReadTreeToMap 1 failed: %v", err)
	}

	if len(treeMap1) != 4 {
		t.Fatalf("expected 4 files in commit 1 tree, got %d", len(treeMap1))
	}

	readmeBlobHash1 := treeMap1["README.md"].Hash
	mainBlobHash1 := treeMap1["src/main.go"].Hash
	hashBlobHash1 := treeMap1["src/utils/hash.go"].Hash
	docsBlobHash1 := treeMap1["docs/manual.md"].Hash

	// 7. Validar contenido de blobs recuperados
	rawBlob, _, err := repo.Objects.ReadObject(hashBlobHash1)
	if err != nil {
		t.Fatalf("failed reading hash.go blob: %v", err)
	}
	if !strings.Contains(string(rawBlob), "package utils") {
		t.Fatalf("recovered blob data mismatch: %s", string(rawBlob))
	}

	// Recopilar hashes de subtrees en el commit 1
	rootTreeRaw1, _, _ := repo.Objects.ReadObject(commit1.Tree)
	rootTreeObj1, _ := object.DecodeTree(rootTreeRaw1)
	var srcTreeHash1, docsTreeHash1 string
	for _, entry := range rootTreeObj1.Entries {
		if entry.Name == "src" {
			srcTreeHash1 = entry.Hash
		} else if entry.Name == "docs" {
			docsTreeHash1 = entry.Hash
		}
	}

	srcTreeRaw1, _, _ := repo.Objects.ReadObject(srcTreeHash1)
	srcTreeObj1, _ := object.DecodeTree(srcTreeRaw1)
	var utilsTreeHash1 string
	for _, entry := range srcTreeObj1.Entries {
		if entry.Name == "utils" {
			utilsTreeHash1 = entry.Hash
		}
	}

	// 10. Modificar únicamente src/utils/hash.go
	os.WriteFile(hashPath, []byte("package utils\nfunc Hash() { /* cambio */ }"), 0644)

	// Add y Commit 2
	if err := repo.Add([]string{"src/utils/hash.go"}); err != nil {
		t.Fatalf("Add 2 failed: %v", err)
	}

	res2, err := repo.Commit("Modificar hash.go", "Tester", "test@minigit.local")
	if err != nil {
		t.Fatalf("Commit 2 failed: %v", err)
	}

	commit2, _, err := repo.GetCommitByHash(res2.Hash)
	if err != nil {
		t.Fatalf("GetCommitByHash 2 failed: %v", err)
	}

	treeMap2, err := repo.ReadTreeToMap(commit2.Tree)
	if err != nil {
		t.Fatalf("ReadTreeToMap 2 failed: %v", err)
	}

	hashBlobHash2 := treeMap2["src/utils/hash.go"].Hash
	readmeBlobHash2 := treeMap2["README.md"].Hash
	mainBlobHash2 := treeMap2["src/main.go"].Hash
	docsBlobHash2 := treeMap2["docs/manual.md"].Hash

	rootTreeRaw2, _, _ := repo.Objects.ReadObject(commit2.Tree)
	rootTreeObj2, _ := object.DecodeTree(rootTreeRaw2)
	var srcTreeHash2, docsTreeHash2 string
	for _, entry := range rootTreeObj2.Entries {
		if entry.Name == "src" {
			srcTreeHash2 = entry.Hash
		} else if entry.Name == "docs" {
			docsTreeHash2 = entry.Hash
		}
	}

	srcTreeRaw2, _, _ := repo.Objects.ReadObject(srcTreeHash2)
	srcTreeObj2, _ := object.DecodeTree(srcTreeRaw2)
	var utilsTreeHash2 string
	for _, entry := range srcTreeObj2.Entries {
		if entry.Name == "utils" {
			utilsTreeHash2 = entry.Hash
		}
	}

	// 12. Confirmar propagación de hashes
	if hashBlobHash1 == hashBlobHash2 {
		t.Fatalf("modified blob should have a new hash")
	}
	if utilsTreeHash1 == utilsTreeHash2 {
		t.Fatalf("subtree src/utils should have a new hash")
	}
	if srcTreeHash1 == srcTreeHash2 {
		t.Fatalf("subtree src should have a new hash")
	}
	if commit1.Tree == commit2.Tree {
		t.Fatalf("root tree should have a new hash")
	}
	if res1.Hash == res2.Hash {
		t.Fatalf("commit 2 should have a new hash")
	}

	// 17. Confirmar inmutabilidad de objetos no modificados
	if readmeBlobHash1 != readmeBlobHash2 {
		t.Fatalf("unmodified README.md blob hash should remain identical")
	}
	if mainBlobHash1 != mainBlobHash2 {
		t.Fatalf("unmodified main.go blob hash should remain identical")
	}
	if docsBlobHash1 != docsBlobHash2 {
		t.Fatalf("unmodified docs/manual.md blob hash should remain identical")
	}
	if docsTreeHash1 != docsTreeHash2 {
		t.Fatalf("unmodified docs tree hash should remain identical")
	}

	// 18. Confirmar puntero al padre
	if commit2.Parent != res1.Hash {
		t.Fatalf("commit 2 parent should point to commit 1 (%s), got %s", res1.Hash, commit2.Parent)
	}

	// Validar grafo del commit 2
	if err := repo.ValidateCommitGraph(res2.Hash); err != nil {
		t.Fatalf("ValidateCommitGraph failed for commit 2: %v", err)
	}
}

func TestMerkleGraphValidationErrors(t *testing.T) {
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)

	// Validar árbol con referencia a objeto inexistente
	badEntry := object.TreeEntry{
		Name: "missing.txt",
		Type: "blob",
		Mode: 0644,
		Hash: "9999999999999999999999999999999999999999999999999999999999999999",
	}
	badTree := object.NewTree([]object.TreeEntry{badEntry})
	badTreeHash, err := repo.Objects.WriteObject(badTree.Serialize())
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	err = repo.ValidateTreeRecursively(badTreeHash, nil, 1)
	if err == nil {
		t.Fatalf("expected error for tree referencing non-existent object")
	}

	// Validar inconsistencia de tipo: entrada declarada blob apuntando a tree
	validSubTree := object.NewTree(nil)
	validSubTreeHash, _ := repo.Objects.WriteObject(validSubTree.Serialize())

	mismatchedEntry := object.TreeEntry{
		Name: "wrong_type.txt",
		Type: "blob", // Declarado como blob pero hash es un tree
		Mode: 0644,
		Hash: validSubTreeHash,
	}
	mismatchedTree := object.NewTree([]object.TreeEntry{mismatchedEntry})
	mismatchedTreeHash, _ := repo.Objects.WriteObject(mismatchedTree.Serialize())

	err = repo.ValidateTreeRecursively(mismatchedTreeHash, nil, 1)
	if err == nil || !strings.Contains(err.Error(), "type mismatch") {
		t.Fatalf("expected type mismatch error, got: %v", err)
	}
}

func TestCommitMessageAndAuthorValidation(t *testing.T) {
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)

	os.WriteFile(filepath.Join(repoDir, "a.txt"), []byte("data"), 0644)
	repo.Add([]string{"a.txt"})

	// Mensaje vacío o solo espacios
	_, err := repo.Commit("", "Author", "email@test.com")
	if !errors.Is(err, repository.ErrEmptyCommitMessage) {
		t.Fatalf("expected ErrEmptyCommitMessage for empty message, got: %v", err)
	}

	_, err = repo.Commit("   \n\t  ", "Author", "email@test.com")
	if !errors.Is(err, repository.ErrEmptyCommitMessage) {
		t.Fatalf("expected ErrEmptyCommitMessage for whitespace message, got: %v", err)
	}

	// Commit con autor/email predeterminados
	res, err := repo.Commit("Valid commit", "", "")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	commitObj, _, err := repo.GetCommitByHash(res.Hash)
	if err != nil {
		t.Fatalf("GetCommitByHash failed: %v", err)
	}
	if commitObj.AuthorName != "MiniGit User" || commitObj.AuthorMail != "user@minigit.local" {
		t.Fatalf("expected default author 'MiniGit User <user@minigit.local>', got '%s <%s>'", commitObj.AuthorName, commitObj.AuthorMail)
	}
}

func TestCommitFromIndexNotWorkingTree(t *testing.T) {
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)

	filePath := filepath.Join(repoDir, "file.txt")
	os.WriteFile(filePath, []byte("Content Version A"), 0644)
	repo.Add([]string{"file.txt"})

	// Modificar archivo en Working Tree sin hacer add (Version B)
	os.WriteFile(filePath, []byte("Content Version B (unstaged)"), 0644)

	res, err := repo.Commit("Commit Version A", "Tester", "test@minigit.local")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verificar que el commit contenga "Version A" y no "Version B"
	treeMap, err := repo.ReadTreeToMap(res.RootTreeHash)
	if err != nil {
		t.Fatalf("ReadTreeToMap failed: %v", err)
	}

	fileEntry := treeMap["file.txt"]
	blobRaw, _, err := repo.Objects.ReadObject(fileEntry.Hash)
	if err != nil {
		t.Fatalf("ReadObject failed for blob: %v", err)
	}

	blobObj, err := object.DecodeBlob(blobRaw)
	if err != nil {
		t.Fatalf("DecodeBlob failed: %v", err)
	}

	if string(blobObj.Data) != "Content Version A" {
		t.Fatalf("expected commit blob content to be 'Content Version A', got '%s'", string(blobObj.Data))
	}
}

func TestCommitNoChangesRejection(t *testing.T) {
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)

	os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("content"), 0644)
	repo.Add([]string{"file.txt"})

	res1, err := repo.Commit("Commit 1", "Tester", "test@minigit.local")
	if err != nil {
		t.Fatalf("Commit 1 failed: %v", err)
	}

	// Intentar hacer commit sin modificar nada en el Index
	_, err = repo.Commit("Commit 2 sin cambios", "Tester", "test@minigit.local")
	if !errors.Is(err, repository.ErrNothingToCommit) {
		t.Fatalf("expected ErrNothingToCommit, got: %v", err)
	}

	// Verificar que HEAD no haya cambiado
	headCommit, err := repo.GetHeadCommitHash()
	if err != nil {
		t.Fatalf("GetHeadCommitHash failed: %v", err)
	}
	if headCommit != res1.Hash {
		t.Fatalf("HEAD changed despite rejected commit: expected %s, got %s", res1.Hash, headCommit)
	}
}

func TestCommitDeterminism(t *testing.T) {
	fixedTime := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)

	repoDir1 := t.TempDir()
	repository.InitRepository(repoDir1)
	repo1 := repository.OpenRepository(repoDir1)

	os.WriteFile(filepath.Join(repoDir1, "data.txt"), []byte("same data"), 0644)
	repo1.Add([]string{"data.txt"})

	res1, err := repo1.CommitWithTime("Determinism Test", "Author", "author@test.local", fixedTime)
	if err != nil {
		t.Fatalf("Commit 1 failed: %v", err)
	}

	repoDir2 := t.TempDir()
	repository.InitRepository(repoDir2)
	repo2 := repository.OpenRepository(repoDir2)

	os.WriteFile(filepath.Join(repoDir2, "data.txt"), []byte("same data"), 0644)
	repo2.Add([]string{"data.txt"})

	res2, err := repo2.CommitWithTime("Determinism Test", "Author", "author@test.local", fixedTime)
	if err != nil {
		t.Fatalf("Commit 2 failed: %v", err)
	}

	if res1.Hash != res2.Hash {
		t.Fatalf("commits with identical data & fixed timestamp produced different hashes: %s vs %s", res1.Hash, res2.Hash)
	}
}

func TestCommitErrorCases(t *testing.T) {
	// 1. Lock activo
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)
	os.WriteFile(filepath.Join(repoDir, "a.txt"), []byte("data"), 0644)
	repo.Add([]string{"a.txt"})

	lock, err := repository.AcquireLock(repository.GetIndexPath(repoDir))
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	_, err = repo.Commit("Commit locked", "Author", "author@test.local")
	if !errors.Is(err, repository.ErrLockExists) {
		t.Fatalf("expected ErrLockExists when repo is locked, got: %v", err)
	}
	lock.Unlock()

	// 2. Corrupt Index
	os.WriteFile(repository.GetIndexPath(repoDir), []byte("{invalid json index"), 0644)
	_, err = repo.Commit("Commit corrupt index", "Author", "author@test.local")
	if !errors.Is(err, repository.ErrCorruptIndex) {
		t.Fatalf("expected ErrCorruptIndex, got: %v", err)
	}
}

func TestLogAndInspectObject(t *testing.T) {
	repoDir := t.TempDir()
	repository.InitRepository(repoDir)
	repo := repository.OpenRepository(repoDir)

	// 1. Log en repositorio sin commits
	_, err := repo.GetCommitHistory()
	if !errors.Is(err, repository.ErrNoCommits) {
		t.Fatalf("expected ErrNoCommits for empty repo, got: %v", err)
	}

	// 2. Crear primer commit
	file1Path := filepath.Join(repoDir, "file1.txt")
	os.WriteFile(file1Path, []byte("Content of file 1"), 0644)
	repo.Add([]string{"file1.txt"})
	commit1Res, err := repo.Commit("First commit", "Author1", "author1@test.local")
	if err != nil {
		t.Fatalf("First commit failed: %v", err)
	}

	// 3. Crear segundo commit
	file2Path := filepath.Join(repoDir, "file2.txt")
	os.WriteFile(file2Path, []byte("Content of file 2"), 0644)
	repo.Add([]string{"file2.txt"})
	commit2Res, err := repo.Commit("Second commit", "Author2", "author2@test.local")
	if err != nil {
		t.Fatalf("Second commit failed: %v", err)
	}

	// 4. Log: verificar orden (commit 2 primero, commit 1 segundo)
	history, err := repo.GetCommitHistory()
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 commit log entries, got %d", len(history))
	}
	if history[0].Hash != commit2Res.Hash {
		t.Fatalf("newest commit must be first: expected %s, got %s", commit2Res.Hash, history[0].Hash)
	}
	if history[1].Hash != commit1Res.Hash {
		t.Fatalf("oldest commit must be last: expected %s, got %s", commit1Res.Hash, history[1].Hash)
	}
	if history[1].ParentHash != "" {
		t.Fatalf("root commit must have empty parent hash, got: %s", history[1].ParentHash)
	}

	// 5. InspectObject sobre Commit
	inspectCommit, err := repo.InspectObject(commit2Res.Hash)
	if err != nil {
		t.Fatalf("InspectObject on Commit failed: %v", err)
	}
	if inspectCommit.Type != object.TypeCommit {
		t.Fatalf("expected type commit, got %s", inspectCommit.Type)
	}
	if inspectCommit.Commit.Tree == "" {
		t.Fatalf("expected non-empty tree reference in commit")
	}

	// 6. InspectObject sobre Tree
	inspectTree, err := repo.InspectObject(inspectCommit.Commit.Tree)
	if err != nil {
		t.Fatalf("InspectObject on Tree failed: %v", err)
	}
	if inspectTree.Type != object.TypeTree {
		t.Fatalf("expected type tree, got %s", inspectTree.Type)
	}
	if len(inspectTree.Tree.Entries) == 0 {
		t.Fatalf("tree entries should not be empty")
	}

	// 7. InspectObject sobre Blob
	blobHash := inspectTree.Tree.Entries[0].Hash
	inspectBlob, err := repo.InspectObject(blobHash)
	if err != nil {
		t.Fatalf("InspectObject on Blob failed: %v", err)
	}
	if inspectBlob.Type != object.TypeBlob {
		t.Fatalf("expected type blob, got %s", inspectBlob.Type)
	}
	if string(inspectBlob.BlobData) == "" {
		t.Fatalf("blob data should not be empty")
	}

	// 8. InspectObject sobre objeto inexistente
	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"
	_, err = repo.InspectObject(fakeHash)
	if err == nil || !strings.Contains(err.Error(), "No se encontró el objeto solicitado") {
		t.Fatalf("expected 'No se encontró el objeto solicitado' error, got: %v", err)
	}

	// 9. InspectObject sobre objeto corrupto
	corruptPath := filepath.Join(repoDir, ".minigit", "objects", blobHash[:2], blobHash[2:])
	os.Chmod(corruptPath, 0666)
	os.WriteFile(corruptPath, []byte("invalid corrupt data"), 0666)
	_, err = repo.InspectObject(blobHash)
	if err == nil || !strings.Contains(err.Error(), "Objeto corrupto") {
		t.Fatalf("expected 'Objeto corrupto' error, got: %v", err)
	}
}
