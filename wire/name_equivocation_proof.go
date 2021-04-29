package wire

import (
	"io"

	"fnd/blob"
	"fnd/crypto"

	"fnd.localhost/dwire"
)

type NameEquivocationProof struct {
	HashCacher

	Name string

	RemoteEpochHeight uint16
	RemoteSize        uint16
	RemoteSubdomains  []blob.Subdomain

	LocalEpochHeight uint16
	LocalSize        uint16
	LocalSubdomains  []blob.Subdomain
}

var _ Message = (*NameEquivocationProof)(nil)

func (s *NameEquivocationProof) MsgType() MessageType {
	return MessageTypeNameEquivocationProof
}

func (s *NameEquivocationProof) Equals(other Message) bool {
	cast, ok := other.(*NameEquivocationProof)
	if !ok {
		return false
	}

	return s.Name == cast.Name &&
		s.RemoteEpochHeight == cast.RemoteEpochHeight &&
		s.RemoteSize == cast.RemoteSize &&
		s.LocalEpochHeight == cast.LocalEpochHeight &&
		s.LocalSize == cast.LocalSize
}

func (s *NameEquivocationProof) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		s.Name,
		s.RemoteEpochHeight,
		s.RemoteSize,
		s.RemoteSubdomains,
		s.LocalEpochHeight,
		s.LocalSize,
		s.LocalSubdomains,
	)
}

func (s *NameEquivocationProof) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&s.Name,
		&s.RemoteEpochHeight,
		&s.RemoteSize,
		&s.RemoteSubdomains,
		&s.LocalEpochHeight,
		&s.LocalSize,
		&s.LocalSubdomains,
	)
}

func (s *NameEquivocationProof) Hash() (crypto.Hash, error) {
	return s.HashCacher.Hash(s)
}
