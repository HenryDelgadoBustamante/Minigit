package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrObjectNotFound    = errors.New("object not found")
	ErrAmbiguousHash     = errors.New("short hash is ambiguous")
	ErrCorruptObject     = errors.New("corrupt object: hash mismatch or invalid format")
	ErrInvalidHashFormat = errors.New("invalid hash format")
)

// ObjectStore manages reading and writing content-addressable objects in .minigit/objects/
type ObjectStore struct {
	objectsDir string
}

// NewObjectStore creates a new ObjectStore for a given repository objects directory.
func NewObjectStore(objectsDir string) *ObjectStore {
	return &ObjectStore{objectsDir: objectsDir}
}

// WriteObject writes raw object payload (header + body) into the store using zlib compression.
// Returns the full 64-character SHA-256 hash.
func (s *ObjectStore) WriteObject(rawPayload []byte) (string, error) {
	hash := HashBytes(rawPayload)
	dir := filepath.Join(s.objectsDir, hash[:2])
	filePath := filepath.Join(dir, hash[2:])

	// If file already exists, object is immutable and already saved
	if _, err := os.Stat(filePath); err == nil {
		return hash, nil
	}

	compressed, err := Compress(rawPayload)
	if err != nil {
		return "", fmt.Errorf("compressing object %s failed: %w", hash, err)
	}

	if err := WriteFileAtomic(filePath, compressed, 0444); err != nil {
		return "", fmt.Errorf("writing object file %s failed: %w", hash, err)
	}

	return hash, nil
}

// ReadObject reads and decompresses the object given a full or prefix SHA-256 hash.
// Verifies that the decompressed data matches the hash.
func (s *ObjectStore) ReadObject(hashPrefix string) ([]byte, string, error) {
	fullHash, err := s.ResolveHash(hashPrefix)
	if err != nil {
		return nil, "", err
	}

	dir := filepath.Join(s.objectsDir, fullHash[:2])
	filePath := filepath.Join(dir, fullHash[2:])

	compressed, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("%w: %s", ErrObjectNotFound, fullHash)
		}
		return nil, "", fmt.Errorf("reading object %s failed: %w", fullHash, err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrCorruptObject, err)
	}

	// Verify integrity: recalculated SHA-256 must match expected full hash
	actualHash := HashBytes(decompressed)
	if actualHash != fullHash {
		return nil, "", fmt.Errorf("%w: expected %s, got %s", ErrCorruptObject, fullHash, actualHash)
	}

	return decompressed, fullHash, nil
}

// ResolveHash resolves a 64-character full hash or unambiguous short prefix hash to a full hash.
func (s *ObjectStore) ResolveHash(hashPrefix string) (string, error) {
	hashPrefix = strings.TrimSpace(strings.ToLower(hashPrefix))
	if len(hashPrefix) < 4 || len(hashPrefix) > 64 {
		return "", fmt.Errorf("%w: hash prefix length must be between 4 and 64 hex characters", ErrInvalidHashFormat)
	}

	if len(hashPrefix) == 64 {
		dir := filepath.Join(s.objectsDir, hashPrefix[:2])
		filePath := filepath.Join(dir, hashPrefix[2:])
		if _, err := os.Stat(filePath); err == nil {
			return hashPrefix, nil
		}
		return "", fmt.Errorf("%w: %s", ErrObjectNotFound, hashPrefix)
	}

	// Short hash lookup
	prefixDir := hashPrefix[:2]
	dirPath := filepath.Join(s.objectsDir, prefixDir)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrObjectNotFound, hashPrefix)
		}
		return "", fmt.Errorf("reading object subfolder failed: %w", err)
	}

	suffixPrefix := hashPrefix[2:]
	var matches []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, suffixPrefix) {
			fullHash := prefixDir + name
			if len(fullHash) == 64 {
				matches = append(matches, fullHash)
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("%w: %s", ErrObjectNotFound, hashPrefix)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("%w: prefix '%s' matched multiple objects (%s)", ErrAmbiguousHash, hashPrefix, strings.Join(matches, ", "))
	}

	return matches[0], nil
}

// Exists checks whether an object exists in the object store.
func (s *ObjectStore) Exists(hash string) bool {
	_, err := s.ResolveHash(hash)
	return err == nil
}
