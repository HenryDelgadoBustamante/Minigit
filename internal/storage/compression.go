package storage

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// Compress compresses input data using zlib.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, fmt.Errorf("zlib compression write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zlib compression finish failed: %w", err)
	}
	return buf.Bytes(), nil
}

// Decompress decompresses zlib compressed data.
func Decompress(compressed []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("invalid or corrupt compressed object header: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("decompressing object data failed: %w", err)
	}
	return decompressed, nil
}
