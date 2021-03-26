package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type NameUpdateReq struct {
	HashCacher

	Name        string
	EpochHeight uint16
	SectorSize  uint16
}

var _ Message = (*NameUpdateReq)(nil)

func (n *NameUpdateReq) MsgType() MessageType {
	return MessageTypeNameUpdateReq
}

func (n *NameUpdateReq) Equals(other Message) bool {
	cast, ok := other.(*NameUpdateReq)
	if !ok {
		return false
	}

	return n.Name == cast.Name &&
		n.EpochHeight == cast.EpochHeight &&
		n.SectorSize == cast.SectorSize
}

func (n *NameUpdateReq) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		n.Name,
		n.EpochHeight,
		n.SectorSize,
	)
}

func (n *NameUpdateReq) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&n.Name,
		&n.EpochHeight,
		&n.SectorSize,
	)
}

func (n *NameUpdateReq) Hash() (crypto.Hash, error) {
	return n.HashCacher.Hash(n)
}
