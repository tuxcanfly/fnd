package blob

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"fnd/crypto"
	"io"

	"github.com/btcsuite/btcd/btcec"
)

type Subdomain struct {
	ID          uint8            `json:"id"`
	Name        string           `json:"name"`
	EpochHeight uint16           `json:"epoch_height"`
	Size        uint8            `json:"size"`
	PublicKey   *btcec.PublicKey `json:"public_key"`
	Signature   crypto.Signature `json:"signature"`
}

func (s *Subdomain) String() string {
	return fmt.Sprintf("%v: %v, %v, %v\n", s.ID, s.Name, s.EpochHeight, s.Size)
}

func (s *Subdomain) Equals(o *Subdomain) bool {
	return s.Name == o.Name &&
		s.EpochHeight == o.EpochHeight &&
		s.Size == o.Size &&
		s.PublicKey.IsEqual(o.PublicKey) &&
		s.Signature == o.Signature
}

func (s *Subdomain) MarshalJSON() ([]byte, error) {
	out := &struct {
		ID          uint8  `json:"id"`
		Name        string `json:"name"`
		EpochHeight uint16 `json:"epoch_height"`
		Size        uint8  `json:"size"`
		PublicKey   string `json:"public_key"`
		Signature   string `json:"signature"`
	}{
		s.ID,
		s.Name,
		s.EpochHeight,
		s.Size,
		hex.EncodeToString(s.PublicKey.SerializeCompressed()),
		s.Signature.String(),
	}

	return json.Marshal(out)
}

func (s *Subdomain) UnmarshalJSON(b []byte) error {
	in := &struct {
		ID          uint8  `json:"id"`
		Name        string `json:"name"`
		EpochHeight uint16 `json:"epoch_height"`
		Size        uint8  `json:"size"`
		PublicKey   string `json:"public_key"`
		Signature   string `json:"signature"`
	}{}
	if err := json.Unmarshal(b, in); err != nil {
		return err
	}
	sigB, err := hex.DecodeString(in.Signature)
	if err != nil {
		return err
	}
	sig, err := crypto.NewSignatureFromBytes(sigB)
	if err != nil {
		return err
	}

	s.ID = in.ID
	s.Name = in.Name
	s.EpochHeight = in.EpochHeight
	s.Size = in.Size
	s.PublicKey = mustDecodePublicKey(in.PublicKey)
	s.Signature = sig
	return nil
}

func mustDecodePublicKey(in string) *btcec.PublicKey {
	data, err := hex.DecodeString(in)
	if err != nil {
		panic(err)
	}
	pub, err := btcec.ParsePubKey(data, btcec.S256())
	if err != nil {
		panic(err)
	}
	return pub
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
	err = binary.Write(w, binary.BigEndian, s.Signature)
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
	signatureByes := make([]byte, 65)
	err = binary.Read(r, binary.BigEndian, signatureByes)
	if err != nil {
		return err
	}
	signature, err := crypto.NewSignatureFromBytes(signatureByes)
	if err != nil {
		return err
	}
	s.Signature = signature
	return nil
}
