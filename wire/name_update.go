package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type NameUpdate struct {
	HashCacher

	Name        string
	EpochHeight uint16
	SectorSize  uint16
}

var _ Message = (*NameUpdate)(nil)

func (u *NameUpdate) MsgType() MessageType {
	return MessageTypeNameUpdate
}

func (u *NameUpdate) Equals(other Message) bool {
	cast, ok := other.(*NameUpdate)
	if !ok {
		return false
	}

	return u.Name == cast.Name &&
		u.EpochHeight == cast.EpochHeight &&
		u.SectorSize == cast.SectorSize
}

func (u *NameUpdate) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		u.Name,
		u.EpochHeight,
		u.SectorSize,
	)
}

func (u *NameUpdate) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&u.Name,
		&u.EpochHeight,
		&u.SectorSize,
	)
}

func (u *NameUpdate) Hash() (crypto.Hash, error) {
	return u.HashCacher.Hash(u)
}
