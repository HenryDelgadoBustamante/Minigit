package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeFastForwardSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "minigit-merge-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := InitRepository(tempDir)
	if err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	// 1. Initial Commit on main
	file1Path := filepath.Join(tempDir, "base.txt")
	if err := os.WriteFile(file1Path, []byte("base content\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := repo.Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	res1, err := repo.Commit("Commit 1: base", "Dev", "dev@org.com")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// 2. Create feature branch and advance it
	if err := repo.CreateBranch("feature", res1.Hash); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}
	if _, err := repo.Checkout("feature"); err != nil {
		t.Fatalf("Checkout feature failed: %v", err)
	}

	featFilePath := filepath.Join(tempDir, "feature.txt")
	if err := os.WriteFile(featFilePath, []byte("feature content\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := repo.Add([]string{"."}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	res2, err := repo.Commit("Commit 2: feature add", "Dev", "dev@org.com")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// 3. Switch back to main and merge feature (Fast-Forward)
	if _, err := repo.Checkout("main"); err != nil {
		t.Fatalf("Checkout main failed: %v", err)
	}

	if err := repo.MergeFastForward("feature"); err != nil {
		t.Fatalf("MergeFastForward failed: %v", err)
	}

	// 4. Verify main commit updated to res2.Hash
	mainCommitHash, err := ReadBranchCommit(tempDir, "main")
	if err != nil {
		t.Fatalf("ReadBranchCommit main failed: %v", err)
	}
	if mainCommitHash != res2.Hash {
		t.Errorf("expected main commit hash %s, got %s", res2.Hash, mainCommitHash)
	}

	// 5. Verify feature.txt exists in working tree
	if _, err := os.Stat(featFilePath); err != nil {
		t.Errorf("feature.txt should exist after fast-forward merge")
	}
}

func TestMergeDivergentBranchesRejection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "minigit-merge-div-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := InitRepository(tempDir)
	if err != nil {
		t.Fatalf("InitRepository failed: %v", err)
	}

	// 1. Base commit
	file1Path := filepath.Join(tempDir, "base.txt")
	os.WriteFile(file1Path, []byte("base content\n"), 0644)
	repo.Add([]string{"."})
	res1, _ := repo.Commit("Commit 1: base", "Dev", "dev@org.com")

	// 2. Create feature branch and commit on feature
	repo.CreateBranch("feature", res1.Hash)
	repo.Checkout("feature")

	featFilePath := filepath.Join(tempDir, "feature.txt")
	os.WriteFile(featFilePath, []byte("feature content\n"), 0644)
	repo.Add([]string{"."})
	repo.Commit("Commit 2 on feature", "Dev", "dev@org.com")

	// 3. Switch to main and commit on main (creating a divergence!)
	repo.Checkout("main")
	mainFilePath := filepath.Join(tempDir, "main_only.txt")
	os.WriteFile(mainFilePath, []byte("main content\n"), 0644)
	repo.Add([]string{"."})
	repo.Commit("Commit 3 on main", "Dev", "dev@org.com")

	// 4. Attempt merge feature into main -> should be rejected!
	err = repo.MergeFastForward("feature")
	if err == nil {
		t.Fatalf("expected error merging divergent branches, got success")
	}

	if !errors.Is(err, ErrDivergentBranches) {
		t.Errorf("expected ErrDivergentBranches, got %v", err)
	}
}
