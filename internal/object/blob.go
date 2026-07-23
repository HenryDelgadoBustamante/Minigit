package object

import (
	"fmt"
)

// Blob represents file content in the object store.
type Blob struct {
	Data []byte
}

// NewBlob creates a new Blob object wrapping raw file bytes.
func NewBlob(data []byte) *Blob {
	if data == nil {
		data = []byte{}
	}
	buf := make([]byte, len(data))
	copy(buf, data)
	return &Blob{Data: buf}
}

// Type returns ObjectType "blob".
func (b *Blob) Type() ObjectType {
	return TypeBlob
}

// Serialize encodes the blob into standard header + body: "blob <size>\x00<data>".
func (b *Blob) Serialize() []byte {
	return EncodeObject(TypeBlob, b.Data)
}

// DecodeBlob parses raw object bytes and returns a Blob struct.
func DecodeBlob(raw []byte) (*Blob, error) {
	objType, _, body, err := DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if objType != TypeBlob {
		return nil, fmt.Errorf("%w: expected blob, got %s", ErrTypeMismatch, objType)
	}
	buf := make([]byte, len(body))
	copy(buf, body)
	return &Blob{Data: buf}, nil
}
