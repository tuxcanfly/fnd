package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type BlobUpdate struct {
	HashCacher

	Name        string
	EpochHeight uint16
	SectorSize  uint16
}

var _ Message = (*BlobUpdate)(nil)

func (u *BlobUpdate) MsgType() MessageType {
	return MessageTypeBlobUpdate
}

func (u *BlobUpdate) Equals(other Message) bool {
	cast, ok := other.(*BlobUpdate)
	if !ok {
		return false
	}

	return u.Name == cast.Name &&
		u.EpochHeight == cast.EpochHeight &&
		u.SectorSize == cast.SectorSize
}

func (u *BlobUpdate) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		u.Name,
		u.EpochHeight,
		u.SectorSize,
	)
}

func (u *BlobUpdate) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&u.Name,
		&u.EpochHeight,
		&u.SectorSize,
	)
}

func (u *BlobUpdate) Hash() (crypto.Hash, error) {
	return u.HashCacher.Hash(u)
}
