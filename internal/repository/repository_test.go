package repository_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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
