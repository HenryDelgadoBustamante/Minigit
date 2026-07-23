package storage

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashBytes computes the SHA-256 checksum of the given byte slice and returns it as a hexadecimal string.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
