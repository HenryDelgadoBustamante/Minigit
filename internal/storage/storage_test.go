package storage_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"minigit/internal/storage"
)

func TestHashConsistencyAndDeterminism(t *testing.T) {
	data1 := []byte("minigit deterministic content 1")
	data2 := []byte("minigit deterministic content 1")
	data3 := []byte("minigit deterministic content 2")

	hash1 := storage.HashBytes(data1)
	hash2 := storage.HashBytes(data2)
	hash3 := storage.HashBytes(data3)

	if len(hash1) != 64 {
		t.Fatalf("expected 64 character SHA-256 hash, got %d", len(hash1))
	}
	if hash1 != hash2 {
		t.Fatalf("identical content must produce identical hash, got %s and %s", hash1, hash2)
	}
	if hash1 == hash3 {
		t.Fatalf("different content must produce different hash, got %s for both", hash1)
	}
}

func TestCompressionAndDecompression(t *testing.T) {
	data := []byte("hello world minigit test data with zlib compression")
	compressed, err := storage.Compress(data)
	if err != nil {
		t.Fatalf("compression failed: %v", err)
	}

	decompressed, err := storage.Decompress(compressed)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}

	if string(decompressed) != string(data) {
		t.Fatalf("expected '%s', got '%s'", string(data), string(decompressed))
	}

	// Empty data decompression test
	_, err = storage.Decompress([]byte{})
	if err == nil {
		t.Fatalf("expected error decompressing empty byte slice")
	}
}

func TestObjectStoreStorageAndDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte("blob 11\x00hello world")
	hash1, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	// Check object existence and pathing structure (.minigit/objects/xx/yyyy...)
	filePath := filepath.Join(tmpDir, hash1[:2], hash1[2:])
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected object file to exist at %s", filePath)
	}

	// Write identical object (deduplication test)
	hash2, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject duplicate write failed: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("duplicate write must return same hash: %s vs %s", hash1, hash2)
	}

	// Read object back with full hash
	readData, fullHash, err := store.ReadObject(hash1)
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}
	if fullHash != hash1 {
		t.Fatalf("expected hash %s, got %s", hash1, fullHash)
	}
	if string(readData) != string(payload) {
		t.Fatalf("expected '%s', got '%s'", string(payload), string(readData))
	}

	// Read object with short hash prefix
	readDataShort, _, err := store.ReadObject(hash1[:8])
	if err != nil {
		t.Fatalf("ReadObject with short hash failed: %v", err)
	}
	if string(readDataShort) != string(payload) {
		t.Fatalf("short hash read mismatch")
	}

	if !store.Exists(hash1) {
		t.Fatalf("store.Exists returned false for existing hash %s", hash1)
	}
}

func TestNonExistentAndInvalidHashHandling(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	// Non-existent 64-char hash
	_, _, err := store.ReadObject("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got: %v", err)
	}

	// Non-existent short prefix
	_, _, err = store.ReadObject("abcd")
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound for missing short hash, got: %v", err)
	}

	// Invalid hash length (too short)
	_, _, err = store.ReadObject("123")
	if !errors.Is(err, storage.ErrInvalidHashFormat) {
		t.Fatalf("expected ErrInvalidHashFormat for short prefix < 4 chars, got: %v", err)
	}

	// Invalid characters (non-hex)
	_, _, err = store.ReadObject("zzzz1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab")
	if !errors.Is(err, storage.ErrInvalidHashFormat) {
		t.Fatalf("expected ErrInvalidHashFormat for non-hex hash, got: %v", err)
	}
}

func TestCorruptObjectDetection(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte("blob 4\x00test")
	hash, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	filePath := filepath.Join(tmpDir, hash[:2], hash[2:])

	// 1. Test decompression failure corruption
	os.Chmod(filePath, 0666)
	os.WriteFile(filePath, []byte("invalid corrupted zlib data"), 0666)

	_, _, err = store.ReadObject(hash)
	if !errors.Is(err, storage.ErrCorruptObject) {
		t.Fatalf("expected ErrCorruptObject for zlib decompression error, got: %v", err)
	}

	// 2. Test SHA-256 integrity mismatch corruption
	tamperedPayload := []byte("blob 4\x00fake")
	compressedTampered, _ := storage.Compress(tamperedPayload)
	os.WriteFile(filePath, compressedTampered, 0666)

	_, _, err = store.ReadObject(hash)
	if !errors.Is(err, storage.ErrCorruptObject) {
		t.Fatalf("expected ErrCorruptObject for hash mismatch, got: %v", err)
	}
}

func TestObjectStoreRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	testPayloads := [][]byte{
		[]byte("blob 11\x00hello world"),
		[]byte("tree 0\x00"),
		[]byte("commit 100\x00tree abc123\nparent def456\nauthor Test <test@test.com> 2026-07-23T12:00:00Z\n\nTest message"),
		{0x00, 0xFF, 0xFE, 0xFD, 0x12, 0x34},
		bytes.Repeat([]byte("X"), 50000),
	}

	for i, payload := range testPayloads {
		hash, err := store.WriteObject(payload)
		if err != nil {
			t.Fatalf("RoundTrip[%d]: WriteObject failed: %v", i, err)
		}

		readData, fullHash, err := store.ReadObject(hash)
		if err != nil {
			t.Fatalf("RoundTrip[%d]: ReadObject failed: %v", i, err)
		}

		if fullHash != hash {
			t.Fatalf("RoundTrip[%d]: hash mismatch: expected %s, got %s", i, hash, fullHash)
		}

		if string(readData) != string(payload) {
			t.Fatalf("RoundTrip[%d]: payload mismatch", i)
		}

		// Verify with short hash
		readDataShort, _, err := store.ReadObject(hash[:8])
		if err != nil {
			t.Fatalf("RoundTrip[%d]: ReadObject with short hash failed: %v", i, err)
		}
		if string(readDataShort) != string(payload) {
			t.Fatalf("RoundTrip[%d]: short hash read mismatch", i)
		}
	}
}

func TestObjectStoreHashStability(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte("stable content for object store hash test")
	hashes := make([]string, 50)

	for i := 0; i < 50; i++ {
		h, err := store.WriteObject(payload)
		if err != nil {
			t.Fatalf("WriteObject[%d] failed: %v", i, err)
		}
		hashes[i] = h
	}

	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("Hash instability at iteration %d: expected %s, got %s", i, hashes[0], hashes[i])
		}
	}
}

func TestObjectStoreEmptyPayload(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte{}
	hash, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject empty payload failed: %v", err)
	}

	readData, _, err := store.ReadObject(hash)
	if err != nil {
		t.Fatalf("ReadObject empty payload failed: %v", err)
	}
	if len(readData) != 0 {
		t.Fatalf("Expected empty payload, got %d bytes", len(readData))
	}
}

func TestObjectStoreCorruptFileDetection(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte("blob 4\x00test")
	hash, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	filePath := filepath.Join(tmpDir, hash[:2], hash[2:])

	// Truncate file to simulate corruption
	os.Chmod(filePath, 0666)
	os.WriteFile(filePath, []byte("truncated"), 0666)

	_, _, err = store.ReadObject(hash)
	if !errors.Is(err, storage.ErrCorruptObject) {
		t.Fatalf("expected ErrCorruptObject for truncated file, got: %v", err)
	}
}

func TestAtomicWriteAndCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Test successful atomic write
	data := []byte("atomic test data")
	err := storage.WriteFileAtomic(filepath.Join(tmpDir, "test.txt"), data, 0644)
	if err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	readData, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readData) != string(data) {
		t.Fatalf("Atomic write data mismatch")
	}

	// Test no leftover temp files
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if len(e.Name()) > 13 && e.Name()[:13] == ".minigit-tmp-" {
			t.Fatalf("Leftover temp file found: %s", e.Name())
		}
	}

	// Test CleanupTempFiles
	os.WriteFile(filepath.Join(tmpDir, ".minigit-tmp-abc123"), []byte("stale"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".minigit-tmp-def456"), []byte("stale2"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "normal.txt"), []byte("keep"), 0644)

	cleaned, err := storage.CleanupTempFiles(tmpDir)
	if err != nil {
		t.Fatalf("CleanupTempFiles failed: %v", err)
	}
	if cleaned != 2 {
		t.Fatalf("Expected 2 cleaned files, got %d", cleaned)
	}

	// Verify normal file still exists
	if _, err := os.Stat(filepath.Join(tmpDir, "normal.txt")); err != nil {
		t.Fatalf("Normal file should not be deleted")
	}
}
