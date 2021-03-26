package blob

import (
	"encoding/hex"
	"encoding/json"
	"fnd/crypto"

	"github.com/btcsuite/btcd/btcec"
)

type Subdomain struct {
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	EpochHeight  uint16           `json:"epoch_height"`
	Size         uint8            `json:"size"`
	PublicKey    *btcec.PublicKey `json:"public_key"`
	ReservedRoot crypto.Hash      `json:"reserved_root"`
}

func (s *Subdomain) MarshalJSON() ([]byte, error) {
	out := &struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		EpochHeight  uint16 `json:"epoch_height"`
		Size         uint8  `json:"sector_size"`
		PublicKey    string `json:"public_key"`
		ReservedRoot string `json:"reserved_root"`
	}{
		s.ID,
		s.Name,
		s.EpochHeight,
		s.Size,
		hex.EncodeToString(s.PublicKey.SerializeCompressed()),
		s.ReservedRoot.String(),
	}

	return json.Marshal(out)
}

func (s *Subdomain) UnmarshalJSON(b []byte) error {
	in := &struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		EpochHeight  uint16 `json:"epoch_height"`
		Size         uint8  `json:"size"`
		PublicKey    string `json:"public_key"`
		ReservedRoot string `json:"reserved_root"`
	}{}
	if err := json.Unmarshal(b, in); err != nil {
		return err
	}
	rrB, err := hex.DecodeString(in.ReservedRoot)
	if err != nil {
		return err
	}
	rr, err := crypto.NewHashFromBytes(rrB)
	if err != nil {
		return err
	}

	s.ID = in.ID
	s.Name = in.Name
	s.EpochHeight = in.EpochHeight
	s.Size = in.Size
	s.PublicKey = mustDecodePublicKey(in.PublicKey)
	s.ReservedRoot = rr
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
