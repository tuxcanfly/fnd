package protocol

import (
	"sync"
	"time"

	"github.com/ddrp-org/ddrp/blob"
	"github.com/ddrp-org/ddrp/config"
	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/ddrp/log"
	"github.com/ddrp-org/ddrp/p2p"
	"github.com/ddrp-org/ddrp/store"
	"github.com/ddrp-org/ddrp/util"
	"github.com/ddrp-org/ddrp/wire"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrUpdaterAlreadySynchronized   = errors.New("updater already synchronized")
	ErrUpdaterSectorTipHashMismatch = errors.New("updater sector tip hash mismatch")
	ErrNameLocked                   = errors.New("name is locked")

	updaterLogger = log.WithModule("updater")
)

type Updater struct {
	PollInterval time.Duration
	Workers      int
	mux          *p2p.PeerMuxer
	db           *leveldb.DB
	queue        *UpdateQueue
	nameLocker   util.MultiLocker
	bs           blob.Store
	obs          *util.Observable
	quitCh       chan struct{}
	wg           sync.WaitGroup
	lgr          log.Logger
}

func NewUpdater(mux *p2p.PeerMuxer, db *leveldb.DB, queue *UpdateQueue, nameLocker util.MultiLocker, bs blob.Store) *Updater {
	return &Updater{
		PollInterval: config.ConvertDuration(config.DefaultConfig.Tuning.Updater.PollIntervalMS, time.Millisecond),
		Workers:      config.DefaultConfig.Tuning.Updater.Workers,
		mux:          mux,
		db:           db,
		queue:        queue,
		nameLocker:   nameLocker,
		bs:           bs,
		obs:          util.NewObservable(),
		quitCh:       make(chan struct{}),
		lgr:          log.WithModule("updater"),
	}
}

func (u *Updater) Start() error {
	for i := 0; i < u.Workers; i++ {
		u.wg.Add(1)
		go u.runWorker()
	}
	u.wg.Wait()
	return nil
}

func (u *Updater) Stop() error {
	close(u.quitCh)
	u.wg.Wait()
	return nil
}

func (u *Updater) OnUpdateProcessed(hdlr func(item *UpdateQueueItem, err error)) util.Unsubscriber {
	return u.obs.On("update:processed", hdlr)
}

func (u *Updater) runWorker() {
	defer u.wg.Done()

	for {
		timer := time.NewTimer(u.PollInterval)
		select {
		case <-timer.C:
			item := u.queue.Dequeue()
			if item == nil {
				continue
			}

			cfg := &UpdateConfig{
				Mux:        u.mux,
				DB:         u.db,
				NameLocker: u.nameLocker,
				BlobStore:  u.bs,
				Item:       item,
			}
			if err := UpdateBlob(cfg); err != nil {
				u.obs.Emit("update:processed", item, err)
				u.lgr.Error("error processing update", "name", item.Name, "err", err)
				continue
			}
			u.obs.Emit("update:processed", item, nil)
			u.lgr.Info("name updated", "name", item.Name)
		case <-u.quitCh:
			return
		}
	}
}

type UpdateConfig struct {
	Mux        *p2p.PeerMuxer
	DB         *leveldb.DB
	NameLocker util.MultiLocker
	BlobStore  blob.Store
	Item       *UpdateQueueItem
}

func UpdateBlob(cfg *UpdateConfig) error {
	l := updaterLogger.Sub("name", cfg.Item.Name)
	item := cfg.Item
	defer item.Dispose()
	header, err := store.GetHeader(cfg.DB, item.Name)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return errors.Wrap(err, "error getting header")
	}
	if header != nil && header.SectorSize == item.SectorSize {
		return ErrUpdaterAlreadySynchronized
	}

	var prevHash crypto.Hash = blob.ZeroHash
	var epochHeight, sectorSize uint16
	if header != nil {
		epochHeight = header.EpochHeight
		sectorSize = header.SectorSize
		if sectorSize != 0 {
			prevHash = header.SectorTipHash
		}
	}

	if !cfg.NameLocker.TryLock(item.Name) {
		return ErrNameLocked
	}
	defer cfg.NameLocker.Unlock(item.Name)

	bl, err := cfg.BlobStore.Open(item.Name)
	if err != nil {
		return errors.Wrap(err, "error getting blob")
	}
	defer func() {
		if err := bl.Close(); err != nil {
			updaterLogger.Error("error closing blob", "err", err)
		}
	}()

	tx, err := bl.Transaction()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}

	err = SyncSectors(&SyncSectorsOpts{
		Timeout:     DefaultSyncerBlobResTimeout,
		Mux:         cfg.Mux,
		Tx:          tx,
		Peers:       item.PeerIDs,
		EpochHeight: epochHeight,
		SectorSize:  sectorSize,
		Name:        item.Name,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			updaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error during sync")
	}
	tree, err := blob.SerialHash(blob.NewReader(tx), prevHash)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			updaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error calculating new blob sector tip hash")
	}
	if tree.Tip() != item.MerkleRoot {
		if err := tx.Rollback(); err != nil {
			updaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return ErrUpdaterSectorTipHashMismatch
	}

	var sectorsNeeded uint16

	if header == nil {
		sectorsNeeded = item.SectorSize
	} else {
		sectorsNeeded = item.SectorSize - header.SectorSize
	}
	l.Debug(
		"calculated needed sectors",
		"total", sectorsNeeded,
	)

	err = store.WithTx(cfg.DB, func(tx *leveldb.Transaction) error {
		return store.SetHeaderTx(tx, &store.Header{
			Name:         item.Name,
			EpochHeight:  item.EpochHeight,
			SectorSize:   item.SectorSize,
			SectorTipHash:   item.MerkleRoot,
			Signature:    item.Signature,
			ReservedRoot: item.ReservedRoot,
			ReceivedAt:   time.Now(),
		}, blob.ZeroSectorHashes)
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			updaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error storing header")
	}
	tx.Commit()

	height, err := store.GetLastNameImportHeight(cfg.DB)
	if err != nil {
		updaterLogger.Error("error getting last name import height, skipping gossip", "err", err)
		return nil
	}
	if height-item.Height < 10 {
		updaterLogger.Info("updated name is below gossip height, skipping", "name", item.Name)
		return nil
	}

	update := &wire.Update{
		Name:        item.Name,
		EpochHeight: item.EpochHeight,
		SectorSize:  item.SectorSize,
	}
	p2p.GossipAll(cfg.Mux, update)
	return nil
}
