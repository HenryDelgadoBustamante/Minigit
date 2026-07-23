package object_test

import (
	"testing"
	"time"

	"minigit/internal/object"
)

func TestBlobObject(t *testing.T) {
	content := []byte("hello blob")
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
		{Name: "b.txt", Hash: "hash2", Type: "blob", Mode: 0644},
		{Name: "a.txt", Hash: "hash1", Type: "blob", Mode: 0644},
	}
	entries2 := []object.TreeEntry{
		{Name: "a.txt", Hash: "hash1", Type: "blob", Mode: 0644},
		{Name: "b.txt", Hash: "hash2", Type: "blob", Mode: 0644},
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

func TestCommitObject(t *testing.T) {
	fixedTime := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	commit := object.NewCommit("treehash123", "parenthash456", "Author Name", "author@email.com", "Test commit message\nSecond line", fixedTime)

	serialized := commit.Serialize()
	decoded, err := object.DecodeCommit(serialized)
	if err != nil {
		t.Fatalf("DecodeCommit failed: %v", err)
	}

	if decoded.Tree != "treehash123" {
		t.Fatalf("expected tree treehash123, got %s", decoded.Tree)
	}
	if decoded.Parent != "parenthash456" {
		t.Fatalf("expected parent parenthash456, got %s", decoded.Parent)
	}
	if decoded.AuthorName != "Author Name" || decoded.AuthorMail != "author@email.com" {
		t.Fatalf("author mismatch")
	}
	if decoded.Message != "Test commit message\nSecond line" {
		t.Fatalf("message mismatch: %s", decoded.Message)
	}
}
