package object

import (
	"fmt"
)

// Blob represents file content in the object store.
type Blob struct {
	Data []byte
}

// NewBlob creates a new Blob object.
func NewBlob(data []byte) *Blob {
	return &Blob{Data: data}
}

// Type returns Blob type.
func (b *Blob) Type() ObjectType {
	return TypeBlob
}

// Serialize encodes the blob into "<type> <size>\x00<data>".
func (b *Blob) Serialize() []byte {
	return EncodeObject(TypeBlob, b.Data)
}

// DecodeBlob decodes raw payload bytes into a Blob struct.
func DecodeBlob(raw []byte) (*Blob, error) {
	objType, _, body, err := DecodeObject(raw)
	if err != nil {
		return nil, err
	}
	if objType != TypeBlob {
		return nil, fmt.Errorf("%w: expected blob, got %s", ErrTypeMismatch, objType)
	}
	return &Blob{Data: body}, nil
}
