package blob

import (
	"io"

	"github.com/ddrp-org/ddrp/crypto"
	"golang.org/x/crypto/blake2b"
)

// Hash returns sequential hash of the contents of the reader br
func Hash(br io.Reader) (crypto.Hash, error) {
	hasher, _ := blake2b.New256(nil)
	var hash crypto.Hash
	h := hasher.Sum(nil)
	copy(hash[:], h)
	for i := 0; i < SectorCount; i++ {
		hasher.Write(hash[:])
		if _, err := io.CopyN(hasher, br, SectorLen); err != nil {
			return crypto.ZeroHash, err
		}
		h := hasher.Sum(nil)
		copy(hash[:], h)
	}
	return hash, nil
}
