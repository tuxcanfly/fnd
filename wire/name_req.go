package wire

import (
	"io"

	"fnd/crypto"

	"fnd.localhost/dwire"
)

type NameReq struct {
	HashCacher

	Name        string
	EpochHeight uint16
	SectorSize  uint16
}

var _ Message = (*NameReq)(nil)

func (u *NameReq) MsgType() MessageType {
	return MessageTypeNameReq
}

func (u *NameReq) Equals(other Message) bool {
	cast, ok := other.(*NameReq)
	if !ok {
		return false
	}

	return u.Name == cast.Name &&
		u.EpochHeight == cast.EpochHeight &&
		u.SectorSize == cast.SectorSize
}

func (u *NameReq) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		u.Name,
		u.EpochHeight,
		u.SectorSize,
	)
}

func (u *NameReq) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&u.Name,
		&u.EpochHeight,
		&u.SectorSize,
	)
}

func (u *NameReq) Hash() (crypto.Hash, error) {
	return u.HashCacher.Hash(u)
}
