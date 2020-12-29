package protocol

import (
	"time"

	"github.com/ddrp-org/ddrp/blob"
	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/ddrp/p2p"
	"github.com/ddrp-org/ddrp/wire"
	"github.com/pkg/errors"
)

const (
	DefaultSyncerBlobResTimeout = 15 * time.Second
)

var (
	ErrNoTreeBaseCandidates = errors.New("no tree base candidates")
	ErrSyncerNoProgress     = errors.New("sync not progressing")
	ErrSyncerMaxAttempts    = errors.New("reached max sync attempts")
)

type SyncSectorsOpts struct {
	Timeout     time.Duration
	Mux         *p2p.PeerMuxer
	Tx          blob.Transaction
	Peers       *PeerSet
	EpochHeight uint16
	SectorSize  uint16
	Name        string
}

type sectorRes struct {
	peerID crypto.Hash
	msg    *wire.BlobRes
}

type reqdSectorsMap map[uint8][33]byte

func SyncSectors(opts *SyncSectorsOpts) error {
	// l := log.WithModule("sector-syncer").Sub("name", opts.Name)
	// Implement sector hash based sync
	return nil
}
