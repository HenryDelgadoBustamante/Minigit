package storage

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
)

var (
	ErrCompressionFailed   = errors.New("zlib compression failed")
	ErrDecompressionFailed = errors.New("zlib decompression failed")
)

// Compress compresses input byte data using standard zlib compression.
// Guaranteed to be lossless: decompressing the output returns the exact original bytes.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		w.Close()
		return nil, fmt.Errorf("%w: write failed: %v", ErrCompressionFailed, err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("%w: flush failed: %v", ErrCompressionFailed, err)
	}
	return buf.Bytes(), nil
}

// Decompress decompresses zlib-compressed data.
// Returns a clear error if the compressed payload is invalid, corrupt, or truncated.
func Decompress(compressed []byte) ([]byte, error) {
	if len(compressed) == 0 {
		return nil, fmt.Errorf("%w: compressed data is empty", ErrDecompressionFailed)
	}

	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid header: %v", ErrDecompressionFailed, err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: read body failed: %v", ErrDecompressionFailed, err)
	}
	return decompressed, nil
}
