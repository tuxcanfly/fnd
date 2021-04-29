package store

import (
	"bytes"
	"fnd/wire"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	blobEquivocationProofsPrefix = Prefixer("blobequivocationproofs")
	nameEquivocationProofsPrefix = Prefixer("nameequivocationproofs")
)

func GetBlobEquivocationProof(db *leveldb.DB, name string) ([]byte, error) {
	bytes, err := db.Get(blobEquivocationProofsPrefix(name), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting blobEquivocation proof")
	}
	return bytes, nil
}

func SetBlobEquivocationProofTx(tx *leveldb.Transaction, name string, proof *wire.BlobEquivocationProof) error {
	var buf bytes.Buffer
	proof.Encode(&buf)
	err := tx.Put(blobEquivocationProofsPrefix(name), buf.Bytes(), nil)
	if err != nil {
		return errors.Wrap(err, "error setting blobEquivocation proof")
	}
	return nil
}

func GetNameEquivocationProof(db *leveldb.DB, name string) ([]byte, error) {
	bytes, err := db.Get(nameEquivocationProofsPrefix(name), nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting nameEquivocation proof")
	}
	return bytes, nil
}

func SetNameEquivocationProofTx(tx *leveldb.Transaction, name string, proof *wire.NameEquivocationProof) error {
	var buf bytes.Buffer
	proof.Encode(&buf)
	err := tx.Put(nameEquivocationProofsPrefix(name), buf.Bytes(), nil)
	if err != nil {
		return errors.Wrap(err, "error setting nameEquivocation proof")
	}
	return nil
}
