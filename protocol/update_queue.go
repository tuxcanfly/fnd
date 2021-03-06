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

	"fnd.localhost/handshake/primitives"
	"github.com/btcsuite/btcd/btcec"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrUpdateQueueMaxLen        = errors.New("update queue is at max length")
	ErrUpdateQueueEpochUpdated  = errors.New("epoch already updated")
	ErrUpdateQueueSectorUpdated = errors.New("sector already updated")
	ErrUpdateQueueThrottled     = errors.New("update is throttled")
	ErrUpdateQueueStaleSector   = errors.New("sector is stale")
	ErrUpdateQueueSpltBrain     = errors.New("split brain")
	ErrInitialImportIncomplete  = errors.New("initial import incomplete")
)

type UpdateQueue struct {
	MaxLen   int32
	mux      *p2p.PeerMuxer
	db       *leveldb.DB
	entries  map[string]*UpdateQueueItem
	quitCh   chan struct{}
	queue    []string
	queueLen int32
	mu       sync.Mutex
	lgr      log.Logger
}

type UpdateQueueItem struct {
	PeerIDs     *PeerSet
	Name        string
	EpochHeight uint16
	SectorSize  uint16
	Pub         *btcec.PublicKey
	Height      int
	Disposed    int32
}

func (u *UpdateQueueItem) Dispose() {
	atomic.StoreInt32(&u.Disposed, 1)
}

func NewUpdateQueue(mux *p2p.PeerMuxer, db *leveldb.DB) *UpdateQueue {
	return &UpdateQueue{
		MaxLen:  int32(config.DefaultConfig.Tuning.UpdateQueue.MaxLen),
		mux:     mux,
		db:      db,
		entries: make(map[string]*UpdateQueueItem),
		quitCh:  make(chan struct{}),
		lgr:     log.WithModule("update-queue"),
	}
}

func (u *UpdateQueue) Start() error {
	u.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeUpdate, u.onUpdate))
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

func (u *UpdateQueue) Stop() error {
	close(u.quitCh)
	return nil
}

// TODO: prioritize equivocations first, then higher epochs and sector sizes
func (u *UpdateQueue) Enqueue(peerID crypto.Hash, update *wire.Update) error {
	// use atomic below to prevent having to lock mu
	// during expensive name validation calls when
	// we can cheaply check for the queue size.
	if atomic.LoadInt32(&u.queueLen) >= u.MaxLen {
		return ErrUpdateQueueMaxLen
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

	nameInfo, err := store.GetNameInfo(u.db, update.Name)
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
		return ErrUpdateQueueEpochUpdated
	}

	// Ignore updates for sectors below ours, same epoch
	if epochHeight == update.EpochHeight {
		if sectorSize > update.SectorSize {
			return ErrUpdateQueueStaleSector
		}
		if sectorSize == update.SectorSize {
			return ErrUpdateQueueSectorUpdated
		}
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	entry := u.entries[update.Name]
	if entry == nil || entry.SectorSize < update.SectorSize {
		u.entries[update.Name] = &UpdateQueueItem{
			PeerIDs:     NewPeerSet([]crypto.Hash{peerID}),
			Name:        update.Name,
			EpochHeight: update.EpochHeight,
			SectorSize:  update.SectorSize,
			Pub:         nameInfo.PublicKey,
			Height:      nameInfo.ImportHeight,
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
		return ErrUpdateQueueEpochUpdated
	}

	// Ignore updates for sectors below ours, if entry already exists
	if entry.EpochHeight == update.EpochHeight {
		if entry.SectorSize > update.SectorSize {
			return ErrUpdateQueueStaleSector
		}
	}

	u.lgr.Info("enqueued update", "name", update.Name, "epoch", update.EpochHeight, "sector", update.SectorSize)
	entry.PeerIDs.Add(peerID)
	return nil
}

func (u *UpdateQueue) Dequeue() *UpdateQueueItem {
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

func (u *UpdateQueue) onUpdate(peerID crypto.Hash, envelope *wire.Envelope) {
	update := envelope.Message.(*wire.Update)
	if err := u.Enqueue(peerID, update); err != nil {
		u.lgr.Info("update rejected", "name", update.Name, "reason", err)
	}
}

func (u *UpdateQueue) validateUpdate(name string) error {
	if err := primitives.ValidateName(name); err != nil {
		return errors.Wrap(err, "update name is invalid")
	}
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

func (u *UpdateQueue) reapDequeuedUpdates() {
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
