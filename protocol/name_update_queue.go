package protocol

import (
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
	ErrNameUpdateQueueMaxLen        = errors.New("update queue is at max length")
	ErrNameUpdateQueueEpochUpdated  = errors.New("epoch already updated")
	ErrNameUpdateQueueSectorUpdated = errors.New("sector already updated")
	ErrNameUpdateQueueThrottled     = errors.New("update is throttled")
	ErrNameUpdateQueueStaleSector   = errors.New("sector is stale")
	ErrNameUpdateQueueSpltBrain     = errors.New("split brain")
	ErrNameInitialImportIncomplete  = errors.New("initial import incomplete")
)

type NameUpdateQueue struct {
	MaxLen   int32
	mux      *p2p.PeerMuxer
	db       *leveldb.DB
	entries  map[string]*NameUpdateQueueItem
	quitCh   chan struct{}
	queue    []string
	queueLen int32
	mu       sync.Mutex
	lgr      log.Logger
}

type NameUpdateQueueItem struct {
	PeerIDs     *PeerSet
	Name        string
	EpochHeight uint16
	SectorSize  uint16
	Pub         *btcec.PublicKey
	Height      int
	Disposed    int32
}

func (u *NameUpdateQueueItem) Dispose() {
	atomic.StoreInt32(&u.Disposed, 1)
}

func NewNameUpdateQueue(mux *p2p.PeerMuxer, db *leveldb.DB) *NameUpdateQueue {
	return &NameUpdateQueue{
		MaxLen:  int32(config.DefaultConfig.Tuning.UpdateQueue.MaxLen),
		mux:     mux,
		db:      db,
		entries: make(map[string]*NameUpdateQueueItem),
		quitCh:  make(chan struct{}),
		lgr:     log.WithModule("update-queue"),
	}
}

func (u *NameUpdateQueue) Start() error {
	u.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeNameUpdate, u.onUpdate))
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

func (u *NameUpdateQueue) Stop() error {
	close(u.quitCh)
	return nil
}

// TODO: prioritize equivocations first, then higher epochs and sector sizes
func (u *NameUpdateQueue) Enqueue(peerID crypto.Hash, update *wire.NameUpdate) error {
	// use atomic below to prevent having to lock mu
	// during expensive name validation calls when
	// we can cheaply check for the queue size.
	if atomic.LoadInt32(&u.queueLen) >= u.MaxLen {
		return ErrNameUpdateQueueMaxLen
	}

	initialImportComplete, err := store.GetInitialImportComplete(u.db)
	if err != nil {
		return errors.Wrap(err, "error getting initial import complete")
	}
	if !initialImportComplete {
		return ErrNameInitialImportIncomplete
	}

	if err := u.validateUpdate(update.Name); err != nil {
		return err
	}

	nameInfo, err := store.GetNameInfo(u.db, update.Name)
	if err != nil {
		return errors.Wrap(err, "error getting name info")
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	entry := u.entries[update.Name]
	if entry == nil || entry.SectorSize < update.SectorSize {
		u.entries[update.Name] = &NameUpdateQueueItem{
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

	u.lgr.Info("enqueued update", "name", update.Name, "epoch", update.EpochHeight, "sector", update.SectorSize)
	entry.PeerIDs.Add(peerID)
	return nil
}

func (u *NameUpdateQueue) Dequeue() *NameUpdateQueueItem {
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

func (u *NameUpdateQueue) onUpdate(peerID crypto.Hash, envelope *wire.Envelope) {
	update := envelope.Message.(*wire.NameUpdate)
	if err := u.Enqueue(peerID, update); err != nil {
		u.lgr.Info("update rejected", "name", update.Name, "reason", err)
	}
}

func (u *NameUpdateQueue) validateUpdate(name string) error {
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

func (u *NameUpdateQueue) reapDequeuedUpdates() {
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
