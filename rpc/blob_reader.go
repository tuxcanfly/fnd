package rpc

import (
	"context"
	"errors"
	"fnd/blob"
	apiv1 "fnd/rpc/v1"
	"io"
)

type BlobReader struct {
	client apiv1.Footnotev1Client
	name   string
	off    int64
	size   int64
}

func NewBlobReader(client apiv1.Footnotev1Client, name string) *BlobReader {
	res, _ := client.BlobSize(context.Background(), &apiv1.BlobSizeReq{
		Name: name,
	})
	// TODO; handle error
	return &BlobReader{
		client: client,
		name:   name,
		size:   int64(res.Size),
	}
}

func (b *BlobReader) ReadAt(p []byte, off int64) (int, error) {
	res, err := b.client.BlobReadAt(context.Background(), &apiv1.BlobReadAtReq{
		Name:   b.name,
		Offset: uint32(off),
		Len:    uint32(len(p)),
	})
	if err != nil {
		return 0, err
	}
	n := len(res.Data)
	copy(p, res.Data)
	if n != len(p) {
		return n, errors.New("unequal read - should not happen")
	}
	return n, nil
}

func (b *BlobReader) Read(p []byte) (int, error) {
	if b.off == b.size*blob.SectorBytes {
		return 0, io.EOF
	}
	toRead := len(p)
	if b.off+int64(toRead) > b.size*blob.SectorBytes {
		toRead = int(b.size*blob.SectorBytes) - int(b.off)
	}
	n, err := b.ReadAt(p[:toRead], b.off)
	b.off += int64(n)
	return n, err
}

func (b *BlobReader) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if b.off > b.size*blob.SectorBytes {
			return b.off, errors.New("seek beyond blob bounds")
		}
		b.off = off
	case io.SeekCurrent:
		next := b.off + off
		if next > b.size*blob.SectorBytes {
			return b.off, errors.New("seek beyond blob bounds")
		}
		b.off = next
	case io.SeekEnd:
		next := b.size*blob.SectorBytes - off
		if next < 0 {
			return b.off, errors.New("seek beyond blob bounds")
		}
		b.off = next
	default:
		panic("invalid whence")
	}
	return b.off, nil
}
