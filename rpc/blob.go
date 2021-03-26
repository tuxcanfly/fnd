package rpc

import (
	"context"
	"fnd/crypto"
	apiv1 "fnd/rpc/v1"
	"fnd/store"
	"io"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
)

func GetBlobInfo(client apiv1.Footnotev1Client, name string) (*store.BlobInfo, error) {
	return GetBlobInfoContext(context.Background(), client, name)
}

func GetBlobInfoContext(ctx context.Context, client apiv1.Footnotev1Client, name string) (*store.BlobInfo, error) {
	res, err := client.GetBlobInfo(ctx, &apiv1.BlobInfoReq{
		Name: name,
	})
	if err != nil {
		return nil, err
	}
	return parseBlobInfoRes(res)
}

func ListBlobInfo(client apiv1.Footnotev1Client, after string, cb func(info *store.BlobInfo) bool) error {
	return ListBlobInfoContext(context.Background(), client, after, cb)
}

func ListBlobInfoContext(ctx context.Context, client apiv1.Footnotev1Client, start string, cb func(info *store.BlobInfo) bool) error {
	stream, err := client.ListBlobInfo(ctx, &apiv1.ListBlobInfoReq{
		Start: start,
	})
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		parsed, err := parseBlobInfoRes(res)
		if err != nil {
			return err
		}
		if !cb(parsed) {
			return nil
		}
	}
}
func GetNameInfoContext(ctx context.Context, client apiv1.Footnotev1Client, name string) (*store.NameInfo, error) {
	res, err := client.GetNameInfo(ctx, &apiv1.NameInfoReq{
		Name: name,
	})
	if err != nil {
		return nil, err
	}
	return parseNameInfoRes(res)
}

func ListNameInfoContext(ctx context.Context, client apiv1.Footnotev1Client, start string, cb func(info *store.NameInfo) bool) error {
	stream, err := client.ListNameInfo(ctx, &apiv1.ListNameInfoReq{
		Start: start,
	})
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		parsed, err := parseNameInfoRes(res)
		if err != nil {
			return err
		}
		if !cb(parsed) {
			return nil
		}
	}
}

func GetNameInfo(client apiv1.Footnotev1Client, name string) (*store.NameInfo, error) {
	return GetNameInfoContext(context.Background(), client, name)
}

func ListNameInfo(client apiv1.Footnotev1Client, after string, cb func(info *store.NameInfo) bool) error {
	return ListNameInfoContext(context.Background(), client, after, cb)
}

func parseBlobInfoRes(res *apiv1.BlobInfoRes) (*store.BlobInfo, error) {
	pub, err := btcec.ParsePubKey(res.PublicKey, btcec.S256())
	if err != nil {
		return nil, errors.Wrap(err, "error parsing public key")
	}
	sectorTipHash, err := crypto.NewHashFromBytes(res.SectorTipHash)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing sector tip hash")
	}
	reservedRoot, err := crypto.NewHashFromBytes(res.ReservedRoot)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing reserved root")
	}
	sig, err := crypto.NewSignatureFromBytes(res.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing signature")
	}

	return &store.BlobInfo{
		Name:          res.Name,
		PublicKey:     pub,
		ImportHeight:  int(res.ImportHeight),
		EpochHeight:   uint16(res.EpochHeight),
		SectorSize:    uint16(res.SectorSize),
		SectorTipHash: sectorTipHash,
		ReservedRoot:  reservedRoot,
		Signature:     sig,
		ReceivedAt:    time.Unix(int64(res.ReceivedAt), 0),
		BannedAt:      time.Unix(int64(res.BannedAt), 0),
	}, nil
}

func parseNameInfoRes(res *apiv1.NameInfoRes) (*store.NameInfo, error) {
	pub, err := btcec.ParsePubKey(res.PublicKey, btcec.S256())
	if err != nil {
		return nil, errors.Wrap(err, "error parsing public key")
	}

	return &store.NameInfo{
		Name:         res.Name,
		PublicKey:    pub,
		ImportHeight: int(res.ImportHeight),
	}, nil
}
