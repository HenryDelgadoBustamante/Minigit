package object

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

type ObjectType string

const (
	TypeBlob   ObjectType = "blob"
	TypeTree   ObjectType = "tree"
	TypeCommit ObjectType = "commit"
)

var (
	ErrInvalidHeader    = errors.New("invalid object header")
	ErrTypeMismatch     = errors.New("object type mismatch")
	ErrSizeMismatch     = errors.New("object size mismatch")
	ErrDuplicateEntry   = errors.New("duplicate entry in tree object")
	ErrInvalidEntryName = errors.New("invalid entry name in tree object")
	ErrInvalidHash      = errors.New("invalid hash format in object")
)

// Object represents a Git-style immutable object.
type Object interface {
	Type() ObjectType
	Serialize() []byte
}

// EncodeObject wraps serialized body data with object header: "<type> <size>\x00<body>"
func EncodeObject(objType ObjectType, body []byte) []byte {
	header := fmt.Sprintf("%s %d\x00", objType, len(body))
	var buf bytes.Buffer
	buf.WriteString(header)
	buf.Write(body)
	return buf.Bytes()
}

// DecodeObject parses "<type> <size>\x00<body>" and validates size & header.
func DecodeObject(raw []byte) (ObjectType, int64, []byte, error) {
	nullIdx := bytes.IndexByte(raw, 0)
	if nullIdx == -1 {
		return "", 0, nil, ErrInvalidHeader
	}

	header := string(raw[:nullIdx])
	parts := bytes.Split(raw[:nullIdx], []byte{' '})
	if len(parts) != 2 {
		return "", 0, nil, fmt.Errorf("%w: header must be '<type> <size>'", ErrInvalidHeader)
	}

	objType := ObjectType(parts[0])
	switch objType {
	case TypeBlob, TypeTree, TypeCommit:
	default:
		return "", 0, nil, fmt.Errorf("%w: unknown object type '%s'", ErrInvalidHeader, objType)
	}

	size, err := strconv.ParseInt(string(parts[1]), 10, 64)
	if err != nil || size < 0 {
		return "", 0, nil, fmt.Errorf("%w: invalid size in header: %v", ErrInvalidHeader, err)
	}

	body := raw[nullIdx+1:]
	if int64(len(body)) != size {
		return "", 0, nil, fmt.Errorf("%w: header declared %d bytes, body has %d bytes", ErrSizeMismatch, size, len(body))
	}

	_ = header
	return objType, size, body, nil
}
