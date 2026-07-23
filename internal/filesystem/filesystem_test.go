package filesystem_test

import (
	"errors"
	"testing"

	"minigit/internal/filesystem"
)

func TestPathSafetyValidation(t *testing.T) {
	repoRoot := t.TempDir()

	// 1. Path traversal rejection
	_, err := filesystem.ValidateRelativePath("../outside.txt", repoRoot)
	if !errors.Is(err, filesystem.ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal, got %v", err)
	}

	// 2. Subpath traversal rejection
	_, err = filesystem.ValidateRelativePath("sub/../../outside.txt", repoRoot)
	if !errors.Is(err, filesystem.ErrPathTraversal) {
		t.Fatalf("expected ErrPathTraversal, got %v", err)
	}

	// 3. Absolute path rejection
	_, err = filesystem.ValidateRelativePath("/etc/passwd", repoRoot)
	if !errors.Is(err, filesystem.ErrAbsolutePath) {
		t.Fatalf("expected ErrAbsolutePath, got %v", err)
	}

	// 4. Null byte rejection
	_, err = filesystem.ValidateRelativePath("file\x00.txt", repoRoot)
	if !errors.Is(err, filesystem.ErrNullByte) {
		t.Fatalf("expected ErrNullByte, got %v", err)
	}

	// 5. Internal repository path rejection
	_, err = filesystem.ValidateRelativePath(".minigit/config", repoRoot)
	if !errors.Is(err, filesystem.ErrInternalPath) {
		t.Fatalf("expected ErrInternalPath, got %v", err)
	}

	// 6. Valid safe relative path
	norm, err := filesystem.ValidateRelativePath("src/main.go", repoRoot)
	if err != nil {
		t.Fatalf("expected safe path, got error: %v", err)
	}
	if norm != "src/main.go" {
		t.Fatalf("expected 'src/main.go', got '%s'", norm)
	}
}
