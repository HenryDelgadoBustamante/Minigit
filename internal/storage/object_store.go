package storage

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrObjectNotFound    = errors.New("object not found: the requested object does not exist in the repository")
	ErrAmbiguousHash     = errors.New("short hash is ambiguous: prefix matches multiple objects")
	ErrCorruptObject     = errors.New("corrupt object: integrity check failed (hash mismatch or invalid format)")
	ErrInvalidHashFormat = errors.New("invalid hash format: must be 4-64 hexadecimal characters")
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
		return nil, "", fmt.Errorf("failed to read object file %s: %w", fullHash, err)
	}

	if len(compressed) == 0 {
		return nil, "", fmt.Errorf("%w: object file is empty (%s)", ErrCorruptObject, fullHash)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		return nil, "", fmt.Errorf("%w: zlib decompression failed for %s: %v", ErrCorruptObject, fullHash, err)
	}

	// Verify integrity: recalculated SHA-256 must match expected full hash
	actualHash := HashBytes(decompressed)
	if actualHash != fullHash {
		return nil, "", fmt.Errorf("%w: expected %s, got %s", ErrCorruptObject, fullHash, actualHash)
	}

	return decompressed, fullHash, nil
}

// ReadObjectType reads only the header of an object to determine its type without decompressing the full payload.
func (s *ObjectStore) ReadObjectType(hashPrefix string) (string, string, error) {
	fullHash, err := s.ResolveHash(hashPrefix)
	if err != nil {
		return "", "", err
	}

	dir := filepath.Join(s.objectsDir, fullHash[:2])
	filePath := filepath.Join(dir, fullHash[2:])

	f, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("%w: %s", ErrObjectNotFound, fullHash)
		}
		return "", "", fmt.Errorf("failed to open object file %s: %w", fullHash, err)
	}
	defer f.Close()

	zr, err := zlib.NewReader(f)
	if err != nil {
		return "", "", fmt.Errorf("%w: zlib decompression failed for %s: %v", ErrCorruptObject, fullHash, err)
	}
	defer zr.Close()

	var buf [64]byte
	n, _ := io.ReadFull(zr, buf[:])
	if n == 0 {
		return "", "", fmt.Errorf("%w: object file is empty (%s)", ErrCorruptObject, fullHash)
	}

	nullIdx := bytes.IndexByte(buf[:n], 0)
	if nullIdx == -1 {
		return "", "", fmt.Errorf("%w: missing header null byte in %s (invalid object format)", ErrCorruptObject, fullHash)
	}

	parts := bytes.Split(buf[:nullIdx], []byte{' '})
	if len(parts) != 2 {
		return "", "", fmt.Errorf("%w: invalid header format in %s (expected '<type> <size>')", ErrCorruptObject, fullHash)
	}

	return string(parts[0]), fullHash, nil
}

// ResolveHash resolves a 64-character full hash or unambiguous short prefix hash to a full hash.
func (s *ObjectStore) ResolveHash(hashPrefix string) (string, error) {
	hashPrefix = strings.TrimSpace(strings.ToLower(hashPrefix))
	if len(hashPrefix) < 4 || len(hashPrefix) > 64 {
		return "", fmt.Errorf("%w: hash prefix length must be between 4 and 64 hex characters", ErrInvalidHashFormat)
	}

	for _, c := range hashPrefix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return "", fmt.Errorf("%w: hash prefix contains non-hexadecimal character '%c'", ErrInvalidHashFormat, c)
		}
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
