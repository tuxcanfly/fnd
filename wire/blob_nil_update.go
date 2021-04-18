package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type BlobNilUpdate struct {
	HashCacher

	Name string

	hash crypto.Hash
}

var _ Message = (*BlobNilUpdate)(nil)

func NewBlobNilUpdate(name string) *BlobNilUpdate {
	return &BlobNilUpdate{
		Name: name,
	}
}

func (n *BlobNilUpdate) MsgType() MessageType {
	return MessageTypeBlobNilUpdate
}

func (n *BlobNilUpdate) Equals(other Message) bool {
	cast, ok := other.(*BlobNilUpdate)
	if !ok {
		return false
	}
	return n.Name == cast.Name
}

func (n *BlobNilUpdate) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		n.Name,
	)
}

func (n *BlobNilUpdate) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&n.Name,
	)
}

func (n *BlobNilUpdate) Hash() (crypto.Hash, error) {
	return n.HashCacher.Hash(n)
}
