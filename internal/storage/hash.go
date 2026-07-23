package storage

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashBytes computes the SHA-256 checksum of the given byte slice and returns it as a 64-character lowercase hexadecimal string.
// It is deterministic: identical content will always produce the same hash, and different contents will produce distinct hashes.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
