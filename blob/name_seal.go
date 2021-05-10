package blob

import (
	"fnd/crypto"

	"fnd.localhost/dwire"
	"golang.org/x/crypto/blake2b"
)

func NameSealHash(subdomain *Subdomain) crypto.Hash {
	h, _ := blake2b.New256(nil)
	if _, err := h.Write([]byte("FNBLOB")); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, subdomain.ID); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, subdomain.Name); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, subdomain.EpochHeight); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, subdomain.Size); err != nil {
		panic(err)
	}
	if err := dwire.EncodeField(h, subdomain.PublicKey); err != nil {
		panic(err)
	}

	var out crypto.Hash
	copy(out[:], h.Sum(nil))
	return out
}

func NameSignSeal(signer crypto.Signer, subdomain *Subdomain) (crypto.Signature, error) {
	h := NameSealHash(subdomain)
	return signer.Sign(h)
}
