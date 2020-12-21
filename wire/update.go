package wire

import (
	"io"
	"time"

	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/dwire"
)

type Update struct {
	HashCacher

	Name          string
	Timestamp     time.Time
	SectorTipHash crypto.Hash
	ReservedRoot  crypto.Hash
	Signature     crypto.Signature
}

var _ Message = (*Update)(nil)

func (u *Update) MsgType() MessageType {
	return MessageTypeUpdate
}

func (u *Update) Equals(other Message) bool {
	cast, ok := other.(*Update)
	if !ok {
		return false
	}

	return u.Name == cast.Name &&
		u.Timestamp.Equal(cast.Timestamp) &&
		u.SectorTipHash == cast.SectorTipHash &&
		u.ReservedRoot == cast.ReservedRoot &&
		u.Signature == cast.Signature
}

func (u *Update) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		u.Name,
		u.Timestamp,
		u.SectorTipHash,
		u.ReservedRoot,
		u.Signature,
	)
}

func (u *Update) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&u.Name,
		&u.Timestamp,
		&u.SectorTipHash,
		&u.ReservedRoot,
		&u.Signature,
	)
}

func (u *Update) Hash() (crypto.Hash, error) {
	return u.HashCacher.Hash(u)
}
