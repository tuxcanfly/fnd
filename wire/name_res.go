package wire

import (
	"io"

	"fnd/blob"
	"fnd/crypto"

	"fnd.localhost/dwire"
)

type NameRes struct {
	HashCacher

	Name        string
	EpochHeight uint16
	Subdomains  []blob.Subdomain
	Signature   crypto.Signature
}

var _ Message = (*NameRes)(nil)

func (s *NameRes) MsgType() MessageType {
	return MessageTypeNameRes
}

func (s *NameRes) Equals(other Message) bool {
	cast, ok := other.(*NameRes)
	if !ok {
		return false
	}

	return s.Name == cast.Name &&
		s.EpochHeight == cast.EpochHeight &&
		//s.Subdomains == cast.Subdomains &&
		s.Signature == cast.Signature
}

func (s *NameRes) Encode(w io.Writer) error {
	return dwire.EncodeFields(
		w,
		s.Name,
		s.EpochHeight,
		s.Subdomains,
		s.Signature,
	)
}

func (s *NameRes) Decode(r io.Reader) error {
	return dwire.DecodeFields(
		r,
		&s.Name,
		&s.EpochHeight,
		&s.Subdomains,
		&s.Signature,
	)
}

func (s *NameRes) Hash() (crypto.Hash, error) {
	return s.HashCacher.Hash(s)
}
