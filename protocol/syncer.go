package protocol

import (
	"time"

	"github.com/ddrp-org/ddrp/blob"
	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/ddrp/log"
	"github.com/ddrp-org/ddrp/p2p"
	"github.com/ddrp-org/ddrp/wire"
	"github.com/pkg/errors"
)

const (
	DefaultSyncerBlobResTimeout = 15 * time.Second
)

var (
	ErrSyncerNoProgress  = errors.New("sync not progressing")
	ErrSyncerMaxAttempts = errors.New("reached max sync attempts")
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

func SyncSectors(opts *SyncSectorsOpts) error {
	lgr := log.WithModule("sector-syncer").Sub("name", opts.Name)
	// Implement sector hash based sync
	sectorReqCh := make(chan uint16)
	sectorResCh := make(chan *sectorRes)
	sectorProcessedCh := make(chan struct{}, 1)
	doneCh := make(chan struct{})
	unsubRes := opts.Mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeBlobRes, func(peerID crypto.Hash, envelope *wire.Envelope) {
		sectorResCh <- &sectorRes{
			peerID: peerID,
			msg:    envelope.Message.(*wire.BlobRes),
		}
	}))

	go func() {
		for {
			iter := opts.Peers.Iterator()
			var sendCount int
			select {
			case id := <-sectorReqCh:
				for {
					peerID, ok := iter()
					if !ok {
						break
					}
					if sendCount == 7 {
						break
					}
					err := opts.Mux.Send(peerID, &wire.BlobReq{
						Name:        opts.Name,
						EpochHeight: opts.EpochHeight,
						SectorSize:  id,
					})
					if err != nil {
						lgr.Warn("error fetching sector from peer, trying another", "peer_id", peerID, "err", err)
						continue
					}
					lgr.Debug(
						"requested sector from peer",
						"peer_id", peerID,
					)
					sendCount++
				}
			case res := <-sectorResCh:
				msg := res.msg
				if msg.Name != opts.Name {
					lgr.Trace("received sector for extraneous name", "other_name", msg.Name, "sector_size", msg.SectorSize)
					continue
				}
				if _, err := opts.Tx.WriteAt(msg.Payload[:], int64(blob.SectorLen*msg.SectorSize)); err != nil {
					lgr.Error("failed to write sector", "sector_size", msg.SectorSize, "err", err)
					continue
				}
				sectorProcessedCh <- struct{}{}
			case <-doneCh:
				return
			}
		}
	}()

sectorLoop:
	for i := opts.SectorSize; i < blob.SectorCount; i++ {
		lgr.Debug("requesting sector", "id", i)
		sectorReqCh <- i
		select {
		case <-sectorProcessedCh:
			lgr.Debug("sector processed")
		case <-time.NewTimer(opts.Timeout).C:
			lgr.Warn("sector request timed out")
			break sectorLoop
		}
	}

	unsubRes()
	close(doneCh)
	return nil
}
