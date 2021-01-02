package wire

import (
	"bytes"
	"io"

	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/ddrp/dwire"
)

type BlobRes struct {
	HashCacher

	SectorSize      uint16
	Name            string
	EpochHeight     uint16
	PayloadPosition uint16
	PrevHash        crypto.Hash
	MessageRoot     crypto.Hash
	Signature       crypto.Signature
	Payload         []byte
}

var _ Message = (*BlobRes)(nil)

func (s *BlobRes) MsgType() MessageType {
	return MessageTypeBlobRes
}

func (s *BlobRes) Equals(other Message) bool {
	cast, ok := other.(*BlobRes)
	if !ok {
		return false
	}

	return s.SectorSize == cast.SectorSize &&
		s.Name == cast.Name &&
		s.EpochHeight == cast.EpochHeight &&
		s.PayloadPosition == cast.PayloadPosition &&
		s.PrevHash == cast.PrevHash &&
		s.MessageRoot == cast.MessageRoot &&
		s.Signature == cast.Signature &&
		bytes.Equal(s.Payload, cast.Payload)
}

func (s *BlobRes) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		s.SectorSize,
		s.Name,
		s.EpochHeight,
		s.PayloadPosition,
		s.PrevHash,
		s.MessageRoot,
		s.Signature,
		s.Payload,
	)
}

func (s *BlobRes) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&s.SectorSize,
		&s.Name,
		&s.EpochHeight,
		&s.PayloadPosition,
		&s.PrevHash,
		&s.MessageRoot,
		&s.Signature,
		&s.Payload,
	)
}

func (s *BlobRes) Hash() (crypto.Hash, error) {
	return s.HashCacher.Hash(s)
}
