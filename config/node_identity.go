package config

import (
	"io/ioutil"
	"path"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

const (
	IdentityFilename = "identity"
	NameFilename     = "name"
)

type Identity struct {
	PrivateKey *btcec.PrivateKey
}

func NewIdentity() *Identity {
	pk, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}

	return &Identity{
		pk,
	}
}

func (n *Identity) MarshalBinary() (data []byte, err error) {
	return n.PrivateKey.Serialize(), nil
}

func (n *Identity) UnmarshalBinary(data []byte) error {
	if len(data) != 32 {
		return errors.New("invalid private key length")
	}

	pk, _ := btcec.PrivKeyFromBytes(btcec.S256(), data)
	n.PrivateKey = pk
	return nil
}

func WriteIdentity(homePath string, id *Identity) error {
	idPath := path.Join(homePath, IdentityFilename)
	data, _ := id.MarshalBinary()
	return ioutil.WriteFile(idPath, data, 0644)
}

func WriteName(homePath string, name string) error {
	namePath := path.Join(homePath, NameFilename)
	return ioutil.WriteFile(namePath, []byte(name), 0644)
}

func ReadNodeIdentity(homePath string) (*Identity, error) {
	idPath := path.Join(homePath, IdentityFilename)
	data, err := ioutil.ReadFile(idPath)
	if err != nil {
		return nil, err
	}

	id := &Identity{}
	err = id.UnmarshalBinary(data)
	return id, err
}

func ReadName(homePath string) (string, error) {
	namePath := path.Join(homePath, NameFilename)
	data, err := ioutil.ReadFile(namePath)
	if err != nil {
		return "", err
	}
	return string(data), err
}
