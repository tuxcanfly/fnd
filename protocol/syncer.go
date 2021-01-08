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
					SectorSize:  opts.SectorSize,
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
			select {
			case res := <-sectorResCh:
				msg := res.msg
				if msg.Name != opts.Name {
					lgr.Trace("received sector for extraneous name", "other_name", msg.Name)
					continue
				}
				if opts.SectorSize != msg.PayloadPosition {
					lgr.Trace("received unexpected payload position", "sector_size", opts.SectorSize, "payload_position", msg.PayloadPosition)
					continue
				}
				// FIXME: TODO: Check tip hash
				for i := msg.PayloadPosition; int(i) < len(msg.Payload); i++ {
					if _, err := opts.Tx.WriteAt(msg.Payload[i][:], int64(i)*blob.SectorLen); err != nil {
						lgr.Error("failed to write sector", "sector_id", i, "err", err)
						continue
					}
				}
				sectorProcessedCh <- struct{}{}
			case <-doneCh:
				return
			}
		}
	}()

sectorLoop:
	for {
		lgr.Debug("requesting sector")
		select {
		case <-sectorProcessedCh:
			lgr.Debug("sector processed")
			break sectorLoop
		case <-time.NewTimer(opts.Timeout).C:
			lgr.Warn("sector request timed out")
			break sectorLoop
		}
	}

	unsubRes()
	close(doneCh)
	return nil
}
