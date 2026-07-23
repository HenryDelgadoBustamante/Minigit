package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"minigit/internal/storage"
)

func TestHashAndCompression(t *testing.T) {
	data := []byte("hello world minigit test data")
	hash := storage.HashBytes(data)
	if len(hash) != 64 {
		t.Fatalf("expected 64 char hash, got %d", len(hash))
	}

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
}

func TestObjectStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := storage.NewObjectStore(tmpDir)

	payload := []byte("blob 11\x00hello world")
	hash, err := store.WriteObject(payload)
	if err != nil {
		t.Fatalf("WriteObject failed: %v", err)
	}

	// Read object back with full hash
	readData, fullHash, err := store.ReadObject(hash)
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}
	if fullHash != hash {
		t.Fatalf("expected hash %s, got %s", hash, fullHash)
	}
	if string(readData) != string(payload) {
		t.Fatalf("expected '%s', got '%s'", string(payload), string(readData))
	}

	// Read object with short hash (test short hash resolution)
	shortHash := hash[:8]
	readDataShort, _, err := store.ReadObject(shortHash)
	if err != nil {
		t.Fatalf("ReadObject with short hash failed: %v", err)
	}
	if string(readDataShort) != string(payload) {
		t.Fatalf("short hash read mismatch")
	}

	// Test non-existent hash
	_, _, err = store.ReadObject("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err == nil {
		t.Fatalf("expected error reading non-existent hash")
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

	// Corrupt object file content on disk
	filePath := filepath.Join(tmpDir, hash[:2], hash[2:])
	os.Chmod(filePath, 0666)
	os.WriteFile(filePath, []byte("invalid corrupted zlib data"), 0666)

	_, _, err = store.ReadObject(hash)
	if err == nil {
		t.Fatalf("expected error reading corrupt object")
	}
}
