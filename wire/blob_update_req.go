package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type BlobUpdateReq struct {
	HashCacher

	Name        string
	EpochHeight uint16
	SectorSize  uint16
}

var _ Message = (*BlobUpdateReq)(nil)

func (n *BlobUpdateReq) MsgType() MessageType {
	return MessageTypeBlobUpdateReq
}

func (n *BlobUpdateReq) Equals(other Message) bool {
	cast, ok := other.(*BlobUpdateReq)
	if !ok {
		return false
	}

	return n.Name == cast.Name &&
		n.EpochHeight == cast.EpochHeight &&
		n.SectorSize == cast.SectorSize
}

func (n *BlobUpdateReq) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		n.Name,
		n.EpochHeight,
		n.SectorSize,
	)
}

func (n *BlobUpdateReq) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&n.Name,
		&n.EpochHeight,
		&n.SectorSize,
	)
}

func (n *BlobUpdateReq) Hash() (crypto.Hash, error) {
	return n.HashCacher.Hash(n)
}
