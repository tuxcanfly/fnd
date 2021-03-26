package blob

import (
	"encoding/binary"
	"fnd/crypto"
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
	err := binary.Write(w, binary.BigEndian, s.ID)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, uint8(len(s.Name)))
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, []byte(s.Name))
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.EpochHeight)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.Size)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.PublicKey.SerializeCompressed())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, s.ReservedRoot)
	if err != nil {
		return err
	}
	return nil
}

func (s *Subdomain) Decode(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, &s.ID)
	if err != nil {
		return err
	}
	var nameLength uint8
	err = binary.Read(r, binary.BigEndian, &nameLength)
	if err != nil {
		return err
	}
	nameBytes := make([]byte, nameLength)
	err = binary.Read(r, binary.BigEndian, &nameBytes)
	if err != nil {
		return err
	}
	s.Name = string(nameBytes)
	err = binary.Read(r, binary.BigEndian, &s.EpochHeight)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &s.Size)
	if err != nil {
		return err
	}
	publicKeyBytes := make([]byte, 33)
	err = binary.Read(r, binary.BigEndian, publicKeyBytes)
	if err != nil {
		return err
	}
	publicKey, err := btcec.ParsePubKey(publicKeyBytes, btcec.S256())
	if err != nil {
		return err
	}
	s.PublicKey = publicKey
	err = binary.Read(r, binary.BigEndian, &s.ReservedRoot)
	if err != nil {
		return err
	}
	return nil
}
