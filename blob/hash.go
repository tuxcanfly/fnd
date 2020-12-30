package blob

import (
	"io"

	"github.com/ddrp-org/ddrp/crypto"
	"golang.org/x/crypto/blake2b"
)

var (
	ZeroHash         crypto.Hash
	ZeroSectorHashes SectorHashes
)

type SectorHashes [SectorCount]crypto.Hash

func (s SectorHashes) Encode(w io.Writer) error {
	for _, h := range s {
		if _, err := w.Write(h[:]); err != nil {
			return err
		}
	}
	return nil
}

func (s *SectorHashes) Decode(r io.Reader) error {
	var res SectorHashes
	var hash crypto.Hash
	for i := 0; i < len(res); i++ {
		if _, err := r.Read(hash[:]); err != nil {
			return err
		}
		res[i] = hash
	}

	*s = res
	return nil
}

func (s SectorHashes) DiffWith(other SectorHashes) []uint8 {
	if s == other {
		return nil
	}

	var out []uint8
	for i := 0; i < len(s); i++ {
		if s[i] != other[i] {
			out = append(out, uint8(i))
		}
	}
	return out
}

func (s SectorHashes) Root() crypto.Hash {
	return s[SectorCount-1]
}

func SerialHashSector(sector Sector, prevHash crypto.Hash) crypto.Hash {
	var res crypto.Hash
	hasher, _ := blake2b.New256(nil)
	hasher.Write(prevHash[:])
	hasher.Write(sector[:])
	h := hasher.Sum(nil)
	copy(res[:], h)
	return res
}

// SerialHash returns serial hash of the contents of the reader br
func SerialHash(br io.Reader, prevHash crypto.Hash) (SectorHashes, error) {
	var res SectorHashes
	hasher, _ := blake2b.New256(nil)
	var hash crypto.Hash = prevHash
	for i := 0; i < SectorCount; i++ {
		res[i] = hash
		hasher.Write(hash[:])
		if _, err := io.CopyN(hasher, br, SectorLen); err != nil {
			return ZeroSectorHashes, err
		}
		h := hasher.Sum(nil)
		copy(hash[:], h)
	}
	return res, nil
}
