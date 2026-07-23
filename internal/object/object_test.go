package object_test

import (
	"errors"
	"testing"
	"time"

	"minigit/internal/object"
)

const (
	validHash1 = "1111111111111111111111111111111111111111111111111111111111111111"
	validHash2 = "2222222222222222222222222222222222222222222222222222222222222222"
)

func TestBlobObject(t *testing.T) {
	content := []byte("hello blob content for minigit object store test")
	blob := object.NewBlob(content)
	serialized := blob.Serialize()

	decoded, err := object.DecodeBlob(serialized)
	if err != nil {
		t.Fatalf("DecodeBlob failed: %v", err)
	}
	if string(decoded.Data) != string(content) {
		t.Fatalf("expected '%s', got '%s'", string(content), string(decoded.Data))
	}
}

func TestTreeSortingAndDeterminism(t *testing.T) {
	entries1 := []object.TreeEntry{
		{Name: "b.txt", Hash: validHash2, Type: "blob", Mode: 0644},
		{Name: "a.txt", Hash: validHash1, Type: "blob", Mode: 0644},
	}
	entries2 := []object.TreeEntry{
		{Name: "a.txt", Hash: validHash1, Type: "blob", Mode: 0644},
		{Name: "b.txt", Hash: validHash2, Type: "blob", Mode: 0644},
	}

	tree1 := object.NewTree(entries1)
	tree2 := object.NewTree(entries2)

	ser1 := tree1.Serialize()
	ser2 := tree2.Serialize()

	if string(ser1) != string(ser2) {
		t.Fatalf("tree serialization is not deterministic")
	}

	decoded, err := object.DecodeTree(ser1)
	if err != nil {
		t.Fatalf("DecodeTree failed: %v", err)
	}
	if len(decoded.Entries) != 2 || decoded.Entries[0].Name != "a.txt" || decoded.Entries[1].Name != "b.txt" {
		t.Fatalf("tree entries not sorted correctly in decoded tree")
	}
}

func TestTreeValidationErrors(t *testing.T) {
	// Invalid entry type
	badTypePayload := object.EncodeObject(object.TypeTree, []byte("100644 invalidtype "+validHash1+" file.txt\n"))
	_, err := object.DecodeTree(badTypePayload)
	if !errors.Is(err, object.ErrInvalidHeader) {
		t.Fatalf("expected ErrInvalidHeader for invalid entry type, got: %v", err)
	}

	// Invalid hash length
	badHashPayload := object.EncodeObject(object.TypeTree, []byte("100644 blob short123 file.txt\n"))
	_, err = object.DecodeTree(badHashPayload)
	if !errors.Is(err, object.ErrInvalidHash) {
		t.Fatalf("expected ErrInvalidHash for short entry hash, got: %v", err)
	}

	// Invalid mode octal
	badModePayload := object.EncodeObject(object.TypeTree, []byte("999999 blob "+validHash1+" file.txt\n"))
	_, err = object.DecodeTree(badModePayload)
	if !errors.Is(err, object.ErrInvalidHeader) {
		t.Fatalf("expected ErrInvalidHeader for invalid mode octal, got: %v", err)
	}

	// Duplicate entry name
	dupEntryPayload := object.EncodeObject(object.TypeTree, []byte("100644 blob "+validHash1+" file.txt\n100644 blob "+validHash2+" file.txt\n"))
	_, err = object.DecodeTree(dupEntryPayload)
	if !errors.Is(err, object.ErrDuplicateEntry) {
		t.Fatalf("expected ErrDuplicateEntry, got: %v", err)
	}

	// Invalid entry name with path traversal / subpaths
	badNamePayload := object.EncodeObject(object.TypeTree, []byte("100644 blob "+validHash1+" ../file.txt\n"))
	_, err = object.DecodeTree(badNamePayload)
	if !errors.Is(err, object.ErrInvalidEntryName) {
		t.Fatalf("expected ErrInvalidEntryName for ../, got: %v", err)
	}
}

func TestBinaryBlobPreservation(t *testing.T) {
	binaryData := []byte{0x00, 0xFF, 0xFE, 0xFD, 0x0D, 0x0A, 0x1A, 0x00, 'a', 'b', 'c'}
	blob := object.NewBlob(binaryData)
	serialized := blob.Serialize()

	decoded, err := object.DecodeBlob(serialized)
	if err != nil {
		t.Fatalf("DecodeBlob binary failed: %v", err)
	}

	if len(decoded.Data) != len(binaryData) {
		t.Fatalf("binary blob length mismatch: expected %d, got %d", len(binaryData), len(decoded.Data))
	}
	for i := range binaryData {
		if decoded.Data[i] != binaryData[i] {
			t.Fatalf("binary byte mismatch at offset %d: expected %x, got %x", i, binaryData[i], decoded.Data[i])
		}
	}
}

func TestCommitObject(t *testing.T) {
	fixedTime := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	commit := object.NewCommit(validHash1, validHash2, "Author Name", "author@email.com", "Test commit message\nSecond line", fixedTime)

	serialized := commit.Serialize()
	decoded, err := object.DecodeCommit(serialized)
	if err != nil {
		t.Fatalf("DecodeCommit failed: %v", err)
	}

	if decoded.Tree != validHash1 {
		t.Fatalf("expected tree %s, got %s", validHash1, decoded.Tree)
	}
	if decoded.Parent != validHash2 {
		t.Fatalf("expected parent %s, got %s", validHash2, decoded.Parent)
	}
	if decoded.AuthorName != "Author Name" || decoded.AuthorMail != "author@email.com" {
		t.Fatalf("author mismatch")
	}
	if decoded.Message != "Test commit message\nSecond line" {
		t.Fatalf("message mismatch: %s", decoded.Message)
	}
}

func TestCommitValidationErrors(t *testing.T) {
	// Missing tree
	noTreeBody := []byte("author Author Name <author@email.com> 2026-07-22T12:00:00Z\n\nCommit without tree")
	noTreePayload := object.EncodeObject(object.TypeCommit, noTreeBody)
	_, err := object.DecodeCommit(noTreePayload)
	if !errors.Is(err, object.ErrInvalidHeader) {
		t.Fatalf("expected ErrInvalidHeader for commit missing tree, got: %v", err)
	}

	// Invalid tree hash (short)
	shortTreeBody := []byte("tree short123\nauthor Author Name <author@email.com> 2026-07-22T12:00:00Z\n\nCommit with short tree")
	shortTreePayload := object.EncodeObject(object.TypeCommit, shortTreeBody)
	_, err = object.DecodeCommit(shortTreePayload)
	if !errors.Is(err, object.ErrInvalidHash) {
		t.Fatalf("expected ErrInvalidHash for short tree hash, got: %v", err)
	}

	// Invalid author format
	badAuthorBody := []byte("tree " + validHash1 + "\nauthor Malformed Author Line\n\nCommit with bad author")
	badAuthorPayload := object.EncodeObject(object.TypeCommit, badAuthorBody)
	_, err = object.DecodeCommit(badAuthorPayload)
	if !errors.Is(err, object.ErrInvalidHeader) {
		t.Fatalf("expected ErrInvalidHeader for malformed author line, got: %v", err)
	}
}

func TestDecodeObjectValidation(t *testing.T) {
	// Missing null byte
	_, _, _, err := object.DecodeObject([]byte("blob 10hello"))
	if err == nil {
		t.Fatalf("expected error for header missing null byte")
	}

	// Unknown object type
	_, _, _, err = object.DecodeObject([]byte("invalidtype 5\x00hello"))
	if err == nil {
		t.Fatalf("expected error for unknown object type")
	}

	// Size mismatch
	_, _, _, err = object.DecodeObject([]byte("blob 100\x00hello"))
	if err == nil {
		t.Fatalf("expected size mismatch error")
	}

	// Negative size
	_, _, _, err = object.DecodeObject([]byte("blob -5\x00hello"))
	if err == nil {
		t.Fatalf("expected invalid size error")
	}
}
