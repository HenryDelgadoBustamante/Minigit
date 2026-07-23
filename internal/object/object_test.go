package object_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"minigit/internal/object"
	"minigit/internal/storage"
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

func TestBlobRoundTrip(t *testing.T) {
	contents := [][]byte{
		{},
		[]byte("hello"),
		bytes.Repeat([]byte("A"), 10000),
		{0x00, 0xFF, 0xFE, 0xFD},
		[]byte("line1\nline2\r\nline3\ttab"),
	}

	for i, content := range contents {
		blob := object.NewBlob(content)
		serialized := blob.Serialize()
		decoded, err := object.DecodeBlob(serialized)
		if err != nil {
			t.Fatalf("RoundTrip[%d]: DecodeBlob failed: %v", i, err)
		}
		if string(decoded.Data) != string(content) {
			t.Fatalf("RoundTrip[%d]: content mismatch", i)
		}
		if len(decoded.Data) != len(content) {
			t.Fatalf("RoundTrip[%d]: size mismatch: expected %d, got %d", i, len(content), len(decoded.Data))
		}
	}
}

func TestTreeRoundTrip(t *testing.T) {
	entries := []object.TreeEntry{
		{Name: "z.txt", Hash: validHash1, Type: "blob", Mode: 0644},
		{Name: "a.txt", Hash: validHash2, Type: "blob", Mode: 0644},
		{Name: "src", Hash: validHash1, Type: "tree", Mode: 0755},
	}

	tree := object.NewTree(entries)
	serialized := tree.Serialize()
	decoded, err := object.DecodeTree(serialized)
	if err != nil {
		t.Fatalf("Tree RoundTrip: DecodeTree failed: %v", err)
	}

	if len(decoded.Entries) != len(entries) {
		t.Fatalf("Tree RoundTrip: entry count mismatch: expected %d, got %d", len(entries), len(decoded.Entries))
	}

	// Verify entries are sorted
	if decoded.Entries[0].Name != "a.txt" || decoded.Entries[1].Name != "src" || decoded.Entries[2].Name != "z.txt" {
		t.Fatalf("Tree RoundTrip: entries not sorted correctly")
	}

	for i, e := range decoded.Entries {
		origIdx := -1
		for j, orig := range entries {
			if orig.Name == e.Name {
				origIdx = j
				break
			}
		}
		if origIdx == -1 {
			t.Fatalf("Tree RoundTrip: entry %d '%s' not found in original", i, e.Name)
		}
		if e.Hash != entries[origIdx].Hash {
			t.Fatalf("Tree RoundTrip: hash mismatch for '%s'", e.Name)
		}
		if e.Type != entries[origIdx].Type {
			t.Fatalf("Tree RoundTrip: type mismatch for '%s'", e.Name)
		}
		if e.Mode != entries[origIdx].Mode {
			t.Fatalf("Tree RoundTrip: mode mismatch for '%s'", e.Name)
		}
	}
}

func TestCommitRoundTrip(t *testing.T) {
	fixedTime := time.Date(2026, 1, 15, 8, 30, 45, 0, time.UTC)

	testCases := []struct {
		tree    string
		parent  string
		author  string
		email   string
		message string
	}{
		{validHash1, "", "Root Author", "root@test.com", "Root commit"},
		{validHash1, validHash2, "Child Author", "child@test.com", "Child commit\n\nWith multiple paragraphs\n\nAnd more text"},
		{validHash1, validHash2, "Unicode Authör", "üñícodé@test.com", "Mensaje en español: ¡Hola mundo!"},
		{validHash1, "", "Empty Msg", "empty@test.com", ""},
	}

	for i, tc := range testCases {
		commit := object.NewCommit(tc.tree, tc.parent, tc.author, tc.email, tc.message, fixedTime)
		serialized := commit.Serialize()
		decoded, err := object.DecodeCommit(serialized)
		if err != nil {
			t.Fatalf("Commit RoundTrip[%d]: DecodeCommit failed: %v", i, err)
		}
		if decoded.Tree != tc.tree {
			t.Fatalf("Commit RoundTrip[%d]: tree mismatch", i)
		}
		if decoded.Parent != tc.parent {
			t.Fatalf("Commit RoundTrip[%d]: parent mismatch", i)
		}
		if decoded.AuthorName != tc.author {
			t.Fatalf("Commit RoundTrip[%d]: author mismatch: expected '%s', got '%s'", i, tc.author, decoded.AuthorName)
		}
		if decoded.AuthorMail != tc.email {
			t.Fatalf("Commit RoundTrip[%d]: email mismatch", i)
		}
		if decoded.Message != tc.message {
			t.Fatalf("Commit RoundTrip[%d]: message mismatch: expected '%s', got '%s'", i, tc.message, decoded.Message)
		}
		if !decoded.CreatedAt.Equal(fixedTime) {
			t.Fatalf("Commit RoundTrip[%d]: time mismatch", i)
		}
	}
}

func TestHashStability(t *testing.T) {
	// Blob hash stability
	content := []byte("stable content for hash test")

	hashes := make([]string, 100)
	for i := 0; i < 100; i++ {
		b := object.NewBlob(content)
		hashes[i] = storage.HashBytes(b.Serialize())
	}

	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("Blob hash instability at iteration %d: expected %s, got %s", i, hashes[0], hashes[i])
		}
	}

	// Tree hash stability
	entries := []object.TreeEntry{
		{Name: "a.txt", Hash: validHash1, Type: "blob", Mode: 0644},
		{Name: "b.txt", Hash: validHash2, Type: "blob", Mode: 0644},
	}
	tree := object.NewTree(entries)
	treeSerialized := tree.Serialize()
	treeHash := storage.HashBytes(treeSerialized)

	for i := 0; i < 100; i++ {
		unsortedEntries := []object.TreeEntry{
			{Name: "b.txt", Hash: validHash2, Type: "blob", Mode: 0644},
			{Name: "a.txt", Hash: validHash1, Type: "blob", Mode: 0644},
		}
		t2 := object.NewTree(unsortedEntries)
		h := storage.HashBytes(t2.Serialize())
		if h != treeHash {
			t.Fatalf("Tree hash instability at iteration %d", i)
		}
	}

	// Commit hash stability
	fixedTime := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	commit := object.NewCommit(validHash1, validHash2, "Author", "author@test.com", "Message", fixedTime)
	commitSerialized := commit.Serialize()
	commitHash := storage.HashBytes(commitSerialized)

	for i := 0; i < 100; i++ {
		c := object.NewCommit(validHash1, validHash2, "Author", "author@test.com", "Message", fixedTime)
		h := storage.HashBytes(c.Serialize())
		if h != commitHash {
			t.Fatalf("Commit hash instability at iteration %d", i)
		}
	}
}

func TestEmptyBlob(t *testing.T) {
	blob := object.NewBlob([]byte{})
	serialized := blob.Serialize()
	decoded, err := object.DecodeBlob(serialized)
	if err != nil {
		t.Fatalf("Empty blob decode failed: %v", err)
	}
	if len(decoded.Data) != 0 {
		t.Fatalf("Empty blob should have 0 bytes, got %d", len(decoded.Data))
	}
	if len(blob.Data) != 0 {
		t.Fatalf("Empty blob data should be empty, got %d bytes", len(blob.Data))
	}
}

func TestTreeWithSubtrees(t *testing.T) {
	leafBlob := object.TreeEntry{Name: "leaf.txt", Hash: validHash1, Type: "blob", Mode: 0644}
	subTree := object.NewTree([]object.TreeEntry{leafBlob})
	subTreeHash := storage.HashBytes(subTree.Serialize())

	rootTree := object.NewTree([]object.TreeEntry{
		{Name: "subdir", Hash: subTreeHash, Type: "tree", Mode: 0755},
		{Name: "root.txt", Hash: validHash2, Type: "blob", Mode: 0644},
	})

	serialized := rootTree.Serialize()
	decoded, err := object.DecodeTree(serialized)
	if err != nil {
		t.Fatalf("Tree with subtrees decode failed: %v", err)
	}
	if len(decoded.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(decoded.Entries))
	}
	if decoded.Entries[0].Name != "root.txt" || decoded.Entries[1].Name != "subdir" {
		t.Fatalf("Entries not sorted correctly")
	}
	if decoded.Entries[1].Type != "tree" {
		t.Fatalf("Second entry should be a tree")
	}
}

func TestCommitRootCommit(t *testing.T) {
	fixedTime := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	commit := object.NewCommit(validHash1, "", "Root Author", "root@test.com", "Initial commit", fixedTime)

	if commit.Parent != "" {
		t.Fatalf("Root commit should have empty parent")
	}

	serialized := commit.Serialize()
	decoded, err := object.DecodeCommit(serialized)
	if err != nil {
		t.Fatalf("Root commit decode failed: %v", err)
	}
	if decoded.Parent != "" {
		t.Fatalf("Decoded root commit should have empty parent, got '%s'", decoded.Parent)
	}
}
