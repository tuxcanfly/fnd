package protocol

import (
	"fnd/blob"
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/wire"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	DefaultNameSyncerBlobResTimeout = 15 * time.Second
)

type nameSyncUpdate struct {
	subdomains []blob.Subdomain
}

type NameSyncSubdomainsOpts struct {
	Timeout     time.Duration
	Mux         *p2p.PeerMuxer
	Peers       *PeerSet
	EpochHeight uint16
	SectorSize  uint16
	Name        string
	DB          *leveldb.DB
}

type namePayloadRes struct {
	peerID crypto.Hash
	msg    *wire.NameRes
}

func NameSyncSubdomains(opts *NameSyncSubdomainsOpts) (*nameSyncUpdate, error) {
	lgr := log.WithModule("payload-syncer").Sub("name", opts.Name)
	errs := make(chan error)
	namePayloadResCh := make(chan *namePayloadRes)
	payloadProcessedCh := make(chan *nameSyncUpdate, 1)
	doneCh := make(chan struct{})
	unsubRes := opts.Mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeNameRes, func(peerID crypto.Hash, envelope *wire.Envelope) {
		namePayloadResCh <- &namePayloadRes{
			peerID: peerID,
			msg:    envelope.Message.(*wire.NameRes),
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
				err := opts.Mux.Send(peerID, &wire.NameReq{
					Name:        opts.Name,
					EpochHeight: opts.EpochHeight,
					SectorSize:  opts.SectorSize,
				})
				if err != nil {
					lgr.Warn("error fetching payload from peer, trying another", "peer_id", peerID, "err", err)
					continue
				}
				lgr.Debug(
					"requested payload from peer",
					"peer_id", peerID,
				)
				sendCount++
			}
			select {
			case res := <-namePayloadResCh:
				msg := res.msg
				if msg.Name != opts.Name {
					lgr.Trace("received payload for extraneous name", "other_name", msg.Name)
					continue
				}
				for _, subdomain := range msg.Subdomains {
					info, err := store.GetNameInfo(opts.DB, msg.Name)
					if err != nil {
						lgr.Trace("error reading name info", "err", err)
						continue
					}
					h := blob.NameSealHash(subdomain.Name, subdomain.EpochHeight, subdomain.Size)
					if !crypto.VerifySigPub(info.PublicKey, subdomain.Signature, h) {
						lgr.Trace("error validating signature", "err", err)
						continue
					}
				}
				payloadProcessedCh <- &nameSyncUpdate{
					subdomains: msg.Subdomains,
				}
			case <-doneCh:
				return
			}
		}
	}()

	var err error
	var su *nameSyncUpdate
	timeout := time.NewTimer(opts.Timeout)
payloadLoop:
	for {
		lgr.Debug("requesting payload")
		select {
		case su = <-payloadProcessedCh:
			lgr.Debug("payload processed")
			break payloadLoop
		case <-timeout.C:
			lgr.Warn("payload request timed out")
			break payloadLoop
		case err = <-errs:
			lgr.Warn("payload syncing failed")
			break payloadLoop
		}
	}

	unsubRes()
	close(doneCh)
	return su, err
}
