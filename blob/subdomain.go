package blob

import "crypto"

type Subdomain struct {
	ID           uint16
	Name         string
	EpochHeight  uint16
	Size         uint8
	PublicKey    crypto.PublicKey
	ReservedRoot crypto.Hash
}
