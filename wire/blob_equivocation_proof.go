package wire

import (
	"io"

	"fnd/blob"
	"fnd/crypto"

	"fnd.localhost/dwire"
)

type BlobEquivocationProof struct {
	HashCacher

	Name string

	RemoteEpochHeight     uint16
	RemotePayloadPosition uint16
	RemotePrevHash        crypto.Hash
	RemoteReservedRoot    crypto.Hash
	RemotePayload         []blob.Sector
	RemoteSignature       crypto.Signature

	LocalEpochHeight   uint16
	LocalSectorSize    uint16
	LocalSectorTipHash crypto.Hash
	LocalReservedRoot  crypto.Hash
	LocalSignature     crypto.Signature
}

var _ Message = (*BlobEquivocationProof)(nil)

func (s *BlobEquivocationProof) MsgType() MessageType {
	return MessageTypeBlobEquivocationProof
}

func (s *BlobEquivocationProof) Equals(other Message) bool {
	cast, ok := other.(*BlobEquivocationProof)
	if !ok {
		return false
	}

	return s.Name == cast.Name &&
		s.RemoteEpochHeight == cast.RemoteEpochHeight &&
		s.RemotePayloadPosition == cast.RemotePayloadPosition &&
		s.RemotePrevHash == cast.RemotePrevHash &&
		s.RemoteReservedRoot == cast.RemoteReservedRoot &&
		s.RemoteSignature == cast.RemoteSignature &&
		s.LocalEpochHeight == cast.LocalEpochHeight &&
		s.LocalSectorSize == cast.LocalSectorSize &&
		s.LocalSectorTipHash == cast.LocalSectorTipHash &&
		s.LocalReservedRoot == cast.LocalReservedRoot &&
		s.LocalSignature == cast.LocalSignature
}

func (s *BlobEquivocationProof) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		s.Name,
		s.RemoteEpochHeight,
		s.RemotePayloadPosition,
		s.RemotePrevHash,
		s.RemoteReservedRoot,
		s.RemotePayload,
		s.RemoteSignature,
		s.LocalEpochHeight,
		s.LocalSectorSize,
		s.LocalSectorTipHash,
		s.LocalReservedRoot,
		s.LocalSignature,
	)
}

func (s *BlobEquivocationProof) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&s.Name,
		&s.RemoteEpochHeight,
		&s.RemotePayloadPosition,
		&s.RemotePrevHash,
		&s.RemoteReservedRoot,
		&s.RemotePayload,
		&s.RemoteSignature,
		&s.LocalEpochHeight,
		&s.LocalSectorSize,
		&s.LocalSectorTipHash,
		&s.LocalReservedRoot,
		&s.LocalSignature,
	)
}

func (s *BlobEquivocationProof) Hash() (crypto.Hash, error) {
	return s.HashCacher.Hash(s)
}
