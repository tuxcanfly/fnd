package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type NameNilUpdate struct {
	HashCacher

	Name string

	hash crypto.Hash
}

var _ Message = (*NameNilUpdate)(nil)

func NewNameNilUpdate(name string) *NameNilUpdate {
	return &NameNilUpdate{
		Name: name,
	}
}

func (n *NameNilUpdate) MsgType() MessageType {
	return MessageTypeNameNilUpdate
}

func (n *NameNilUpdate) Equals(other Message) bool {
	cast, ok := other.(*NameNilUpdate)
	if !ok {
		return false
	}
	return n.Name == cast.Name
}

func (n *NameNilUpdate) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		n.Name,
	)
}

func (n *NameNilUpdate) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&n.Name,
	)
}

func (n *NameNilUpdate) Hash() (crypto.Hash, error) {
	return n.HashCacher.Hash(n)
}
