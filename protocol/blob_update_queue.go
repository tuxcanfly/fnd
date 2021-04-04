package protocol

import (
	"fnd/blob"
	"fnd/config"
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/wire"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrBlobUpdateQueueMaxLen        = errors.New("update queue is at max length")
	ErrBlobUpdateQueueEpochUpdated  = errors.New("epoch already updated")
	ErrBlobUpdateQueueSectorUpdated = errors.New("sector already updated")
	ErrBlobUpdateQueueThrottled     = errors.New("update is throttled")
	ErrBlobUpdateQueueStaleSector   = errors.New("sector is stale")
	ErrBlobUpdateQueueSpltBrain     = errors.New("split brain")
	ErrInitialImportIncomplete      = errors.New("initial import incomplete")
)

type BlobUpdateQueue struct {
	MaxLen   int32
	mux      *p2p.PeerMuxer
	db       *leveldb.DB
	entries  map[string]*BlobUpdateQueueItem
	quitCh   chan struct{}
	queue    []string
	queueLen int32
	mu       sync.Mutex
	lgr      log.Logger
}

type BlobUpdateQueueItem struct {
	PeerIDs     *PeerSet
	Name        string
	EpochHeight uint16
	SectorSize  uint16
	Pub         *btcec.PublicKey
	Disposed    int32
}

func (u *BlobUpdateQueueItem) Dispose() {
	atomic.StoreInt32(&u.Disposed, 1)
}

func NewBlobUpdateQueue(mux *p2p.PeerMuxer, db *leveldb.DB) *BlobUpdateQueue {
	return &BlobUpdateQueue{
		MaxLen:  int32(config.DefaultConfig.Tuning.UpdateQueue.MaxLen),
		mux:     mux,
		db:      db,
		entries: make(map[string]*BlobUpdateQueueItem),
		quitCh:  make(chan struct{}),
		lgr:     log.WithModule("update-queue"),
	}
}

func (u *BlobUpdateQueue) Start() error {
	u.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeBlobUpdate, u.onUpdate))
	timer := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-timer.C:
			u.reapDequeuedUpdates()
		case <-u.quitCh:
			return nil
		}
	}
}

func (u *BlobUpdateQueue) Stop() error {
	close(u.quitCh)
	return nil
}

// TODO: prioritize equivocations first, then higher epochs and sector sizes
func (u *BlobUpdateQueue) Enqueue(peerID crypto.Hash, update *wire.BlobUpdate) error {
	// use atomic below to prevent having to lock mu
	// during expensive name validation calls when
	// we can cheaply check for the queue size.
	if atomic.LoadInt32(&u.queueLen) >= u.MaxLen {
		return ErrBlobUpdateQueueMaxLen
	}

	initialImportComplete, err := store.GetInitialImportComplete(u.db)
	if err != nil {
		return errors.Wrap(err, "error getting initial import complete")
	}
	if !initialImportComplete {
		return ErrInitialImportIncomplete
	}

	// An Update with SectorSize zero is a special case i.e.
	// A Equivocation Notification Update. We need to resolve this
	// by requesting an Equivocation Proof by using the special
	// BlobReq Message with SectorSize MaxSectors and handling the
	// subsequent BlobRes (which contains the Equivocation Proof).
	// NOTE: In normal cases, Update with SectorSize zero
	// and BlobReq with SectorSize MaxSectors doesn't make sense.
	if update.SectorSize == 0 {
		err := u.mux.Send(peerID, &wire.BlobReq{
			Name:        update.Name,
			EpochHeight: update.EpochHeight,
			SectorSize:  blob.MaxSectors,
		})
		return err
	}

	if err := u.validateUpdate(update.Name); err != nil {
		return err
	}

	nameInfo, err := store.GetSubdomainInfo(u.db, update.Name)
	if err != nil {
		return errors.Wrap(err, "error getting name info")
	}

	var epochHeight, sectorSize uint16
	header, err := store.GetHeader(u.db, update.Name)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return errors.Wrap(err, "error getting name header")
	} else if err == nil {
		epochHeight = header.EpochHeight
		sectorSize = header.SectorSize
	}

	// Ignore updates for epochs below ours
	if epochHeight > update.EpochHeight {
		return ErrBlobUpdateQueueEpochUpdated
	}

	// Ignore updates for sectors below ours, same epoch
	if epochHeight == update.EpochHeight {
		if sectorSize > update.SectorSize {
			return ErrBlobUpdateQueueStaleSector
		}
		if sectorSize == update.SectorSize {
			return ErrBlobUpdateQueueSectorUpdated
		}
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	entry := u.entries[update.Name]
	if entry == nil || entry.SectorSize < update.SectorSize {
		u.entries[update.Name] = &BlobUpdateQueueItem{
			PeerIDs:     NewPeerSet([]crypto.Hash{peerID}),
			Name:        update.Name,
			EpochHeight: update.EpochHeight,
			SectorSize:  update.SectorSize,
			Pub:         nameInfo.PublicKey,
		}

		if entry == nil {
			u.queue = append(u.queue, update.Name)
			atomic.AddInt32(&u.queueLen, 1)
		}
		u.lgr.Info("enqueued update", "name", update.Name, "epoch", update.EpochHeight, "sector", update.SectorSize)
		return nil
	}

	// Ignore updates for epochs below ours, if entry already exists
	if entry.EpochHeight > update.EpochHeight {
		return ErrBlobUpdateQueueEpochUpdated
	}

	// Ignore updates for sectors below ours, if entry already exists
	if entry.EpochHeight == update.EpochHeight {
		if entry.SectorSize > update.SectorSize {
			return ErrBlobUpdateQueueStaleSector
		}
	}

	u.lgr.Info("enqueued update", "name", update.Name, "epoch", update.EpochHeight, "sector", update.SectorSize)
	entry.PeerIDs.Add(peerID)
	return nil
}

func (u *BlobUpdateQueue) Dequeue() *BlobUpdateQueueItem {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.queue) == 0 {
		return nil
	}

	name := u.queue[0]
	ret := u.entries[name]
	u.queue = u.queue[1:]
	atomic.AddInt32(&u.queueLen, -1)
	delete(u.entries, name)
	return ret
}

func (u *BlobUpdateQueue) onUpdate(peerID crypto.Hash, envelope *wire.Envelope) {
	update := envelope.Message.(*wire.BlobUpdate)
	if err := u.Enqueue(peerID, update); err != nil {
		u.lgr.Info("update rejected", "name", update.Name, "reason", err)
	}
}

func (u *BlobUpdateQueue) validateUpdate(name string) error {
	// TODO: temporarily allow names with "." in them
	// to allow subdomain names to sync
	//if err := primitives.ValidateName(name); err != nil {
	//return errors.Wrap(err, "update name is invalid")
	//}
	nameBan, err := store.NameIsBanned(u.db, name)
	if err != nil {
		return errors.Wrap(err, "error reading name ban state")
	}
	if nameBan {
		return errors.New("name is banned")
	}
	headerBan, err := store.GetHeaderBan(u.db, name)
	if err != nil {
		return errors.Wrap(err, "error reading header ban state")
	}
	if !headerBan.IsZero() {
		return errors.New("header is banned")
	}
	return nil
}

func (u *BlobUpdateQueue) reapDequeuedUpdates() {
	u.mu.Lock()
	defer u.mu.Unlock()
	var toDelete []string
	for k, item := range u.entries {
		if atomic.LoadInt32(&item.Disposed) == 0 {
			continue
		}
		toDelete = append(toDelete, k)
	}
	for _, k := range toDelete {
		delete(u.entries, k)
	}
}
