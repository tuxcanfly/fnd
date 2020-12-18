package wire

import (
	"io"

	"github.com/ddrp-org/ddrp/blob"
	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/dwire"
)

type TreeBaseRes struct {
	HashCacher

	Name         string
	SectorHashes blob.SectorHashes
}

func (d *TreeBaseRes) MsgType() MessageType {
	return MessageTypeTreeBaseRes
}

func (d *TreeBaseRes) Equals(other Message) bool {
	cast, ok := other.(*TreeBaseRes)
	if !ok {
		return false
	}

	return d.Name == cast.Name &&
		d.SectorHashes == cast.SectorHashes
}

func (d *TreeBaseRes) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		d.Name,
		d.SectorHashes,
	)
}

func (d *TreeBaseRes) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&d.Name,
		&d.SectorHashes,
	)
}

func (d *TreeBaseRes) Hash() (crypto.Hash, error) {
	return d.HashCacher.Hash(d)
}
