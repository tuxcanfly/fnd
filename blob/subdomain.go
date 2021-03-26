package blob

import (
	"crypto"
	"io"

	"github.com/btcsuite/btcd/btcec"
)

type Subdomain struct {
	ID           uint16
	Name         string
	EpochHeight  uint16
	Size         uint8
	PublicKey    *btcec.PublicKey
	ReservedRoot crypto.Hash
}

func (s Subdomain) Encode(w io.Writer) error {
	return nil
}

func (s *Subdomain) Decode(r io.Reader) error {
	return nil
}
