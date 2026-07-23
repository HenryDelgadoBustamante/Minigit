package storage_test

import (
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
