package protocol

import (
	"fnd/blob"
	"fnd/config"
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/util"
	"fnd/wire"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrBlobUpdaterAlreadySynchronized     = errors.New("updater already synchronized")
	ErrBlobUpdaterSectorTipHashMismatch   = errors.New("updater sector tip hash mismatch")
	ErrBlobUpdaterLocked                  = errors.New("name is locked")
	ErrBlobUpdaterBanned                  = errors.New("name is banned")
	ErrBlobUpdaterInvalidEpochCurrent     = errors.New("name epoch invalid current")
	ErrBlobUpdaterInvalidEpochThrottled   = errors.New("name epoch invalid throttled")
	ErrBlobUpdaterInvalidEpochBackdated   = errors.New("name epoch invalid backdated")
	ErrBlobUpdaterInvalidEpochFuturedated = errors.New("name epoch invalid futuredated")

	blobUpdaterLogger = log.WithModule("updater")
)

type BlobUpdater struct {
	PollInterval time.Duration
	Workers      int
	mux          *p2p.PeerMuxer
	db           *leveldb.DB
	queue        *BlobUpdateQueue
	nameLocker   util.MultiLocker
	bs           blob.Store
	obs          *util.Observable
	quitCh       chan struct{}
	wg           sync.WaitGroup
	lgr          log.Logger
}

func NewBlobUpdater(mux *p2p.PeerMuxer, db *leveldb.DB, queue *BlobUpdateQueue, nameLocker util.MultiLocker, bs blob.Store) *BlobUpdater {
	return &BlobUpdater{
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

func (u *BlobUpdater) Start() error {
	for i := 0; i < u.Workers; i++ {
		u.wg.Add(1)
		go u.runWorker()
	}
	u.wg.Wait()
	return nil
}

func (u *BlobUpdater) Stop() error {
	close(u.quitCh)
	u.wg.Wait()
	return nil
}

func (u *BlobUpdater) OnUpdateProcessed(hdlr func(item *BlobUpdateQueueItem, err error)) util.Unsubscriber {
	return u.obs.On("update:processed", hdlr)
}

func (u *BlobUpdater) runWorker() {
	defer u.wg.Done()

	for {
		timer := time.NewTimer(u.PollInterval)
		select {
		case <-timer.C:
			item := u.queue.Dequeue()
			if item == nil {
				continue
			}

			cfg := &BlobUpdateConfig{
				Mux:        u.mux,
				DB:         u.db,
				NameLocker: u.nameLocker,
				BlobStore:  u.bs,
				Item:       item,
			}
			if err := BlobUpdateBlob(cfg); err != nil {
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

type BlobUpdateConfig struct {
	Mux        *p2p.PeerMuxer
	DB         *leveldb.DB
	NameLocker util.MultiLocker
	BlobStore  blob.Store
	Item       *BlobUpdateQueueItem
}

func BlobUpdateBlob(cfg *BlobUpdateConfig) error {
	l := blobUpdaterLogger.Sub("name", cfg.Item.Name)
	item := cfg.Item
	defer item.Dispose()
	header, err := store.GetHeader(cfg.DB, item.Name)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return errors.Wrap(err, "error getting header")
	}

	// If the new update and is the same size or fewer reject if it is in the
	// same epoch. In the future, this may be an equivocation condition
	// may not be eq
	if header != nil && header.EpochHeight == item.EpochHeight && header.SectorSize >= item.SectorSize {
		return ErrBlobUpdaterAlreadySynchronized
	}

	var prevHash crypto.Hash = blob.ZeroHash
	var epochHeight, sectorSize uint16
	var epochUpdated bool = true
	if header != nil {
		epochHeight = header.EpochHeight
		sectorSize = header.SectorSize
		prevHash = header.SectorTipHash
		epochUpdated = false
	}

	// header is the existing header/data in the db
	// item is the new incoming update

	// The new header should have a higher or equal epoch
	if item.EpochHeight < epochHeight {
		return ErrBlobUpdaterInvalidEpochBackdated
	}

	bannedAt, err := store.GetHeaderBan(cfg.DB, item.Name)
	if err != nil {
		return err
	}

	// If it is higher (skip if it's appending data to the same epoch)
	if header != nil && item.EpochHeight > epochHeight {
		// Recovery from banning must increment the epoch by at least 2 and one
		// real week since the local node banned
		if !bannedAt.IsZero() {
			// Banned for at least a week
			if bannedAt.Add(7 * 24 * time.Duration(time.Hour)).After(time.Now()) {
				return ErrBlobUpdaterBanned
			}

			// Publisher is banned for the equivocating epoch and the next epoch
			// The faulty epoch may be old or backdated, so the penalty may not be
			// as large as it seems
			if item.EpochHeight <= epochHeight+1 {
				return ErrBlobUpdaterInvalidEpochCurrent
			}
		}

		// If the epoch is updated less than a week ago BUT NOT the current
		// epoch or the next one. The node can bank up one extra epoch just in
		// case (or periodically burst and do two epochs in a week). This
		// conditions is only valid if the last local epoch increment is less
		// than a week old.
		if time.Now().Before(header.EpochStartAt.Add(7 * 24 * time.Duration(time.Hour))) {
			if item.EpochHeight < BlobEpoch(item.Name)+1 {
				return ErrBlobUpdaterInvalidEpochThrottled
			}
		}

		// Reject any epochs more than one in the future
		if item.EpochHeight > BlobEpoch(item.Name)+1 {
			return ErrBlobUpdaterInvalidEpochFuturedated
		}

		// Sync the entire blob on epoch rollover
		epochUpdated = true
		sectorSize = 0
		epochHeight = item.EpochHeight
		prevHash = crypto.ZeroHash
	}

	if !cfg.NameLocker.TryLock(item.Name) {
		return ErrBlobUpdaterLocked
	}
	defer cfg.NameLocker.Unlock(item.Name)

	bl, err := cfg.BlobStore.Open(item.Name)
	if err != nil {
		return errors.Wrap(err, "error getting blob")
	}
	defer func() {
		if err := bl.Close(); err != nil {
			blobUpdaterLogger.Error("error closing blob", "err", err)
		}
	}()

	tx, err := bl.Transaction()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}

	if epochUpdated {
		if epochHeight > BlobEpoch(item.Name)+1 {
			return errors.New("cannot reset epoch ahead of schedule")
		}

		if err := tx.Truncate(); err != nil {
			return err
		}
	}

	_, err = tx.Seek(int64(sectorSize)*int64(blob.SectorBytes), io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "error seeking transaction")
	}

	sectorMeta, err := BlobSyncSectors(&BlobSyncSectorsOpts{
		Timeout:     DefaultBlobSyncerBlobResTimeout,
		Mux:         cfg.Mux,
		Tx:          tx,
		Peers:       item.PeerIDs,
		EpochHeight: epochHeight,
		SectorSize:  sectorSize,
		PrevHash:    prevHash,
		Name:        item.Name,
		DB:          cfg.DB,
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			blobUpdaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error during sync")
	}
	tree, err := blob.SerialHash(blob.NewReader(tx), blob.ZeroHash, item.SectorSize)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			blobUpdaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error calculating new blob sector tip hash")
	}

	if sectorMeta.sectorTipHash != tree.Tip() {
		if err := tx.Rollback(); err != nil {
			blobUpdaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "sector tip hash mismatch")
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

	var epochStart time.Time
	if epochUpdated {
		epochStart = time.Now()
	}

	err = store.WithTx(cfg.DB, func(tx *leveldb.Transaction) error {
		return store.SetHeaderTx(tx, &store.Header{
			Name:          item.Name,
			EpochHeight:   item.EpochHeight,
			SectorSize:    item.SectorSize,
			SectorTipHash: tree.Tip(),
			Signature:     sectorMeta.signature,
			ReservedRoot:  sectorMeta.reservedRoot,
			EpochStartAt:  epochStart,
		}, tree)
	})
	if err != nil {
		if err := tx.Rollback(); err != nil {
			blobUpdaterLogger.Error("error rolling back blob transaction", "err", err)
		}
		return errors.Wrap(err, "error storing header")
	}
	tx.Commit()

	// TODO: revisit this now that we do not have height in update item
	//height, err := store.GetLastNameImportHeight(cfg.DB)
	//if err != nil {
	//blobUpdaterLogger.Error("error getting last name import height, skipping gossip", "err", err)
	//return nil
	//}
	//if height-item.Height < 10 {
	//blobUpdaterLogger.Info("updated name is below gossip height, skipping", "name", item.Name)
	//return nil
	//}

	update := &wire.BlobUpdate{
		Name:        item.Name,
		EpochHeight: item.EpochHeight,
		SectorSize:  item.SectorSize,
	}
	p2p.GossipAll(cfg.Mux, update)
	return nil
}
