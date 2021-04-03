package blob

import (
	"fnd/crypto"

	"fnd.localhost/dwire"
	"golang.org/x/crypto/blake2b"
)

func NameSealHash(name string, epochHeight, size uint16) crypto.Hash {
	h, _ := blake2b.New256(nil)
	if _, err := h.Write([]byte("FNBLOB")); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, name); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, epochHeight); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, size); err != nil {
		panic(err)
	}

	var out crypto.Hash
	copy(out[:], h.Sum(nil))
	return out
}

func NameSignSeal(signer crypto.Signer, name string, epochHeight, size uint16) (crypto.Signature, error) {
	h := NameSealHash(name, epochHeight, size)
	return signer.Sign(h)
}
