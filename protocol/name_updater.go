package protocol

import (
	"fmt"
	"fnd/blob"
	"fnd/config"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/util"
	"fnd/wire"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	ErrNameUpdaterUpdaterAlreadySynchronized   = errors.New("updater already synchronized")
	ErrNameUpdaterUpdaterSectorTipHashMismatch = errors.New("updater sector tip hash mismatch")
	ErrNameUpdaterLocked                       = errors.New("name is locked")
	ErrNameUpdaterBanned                       = errors.New("name is banned")
	ErrNameUpdaterInvalidEpochCurrent          = errors.New("name epoch invalid current")
	ErrNameUpdaterInvalidEpochThrottled        = errors.New("name epoch invalid throttled")
	ErrNameUpdaterInvalidEpochBackdated        = errors.New("name epoch invalid backdated")
	ErrNameUpdaterInvalidEpochFuturedated      = errors.New("name epoch invalid futuredated")

	nameUpdaterLogger = log.WithModule("updater")
)

type NameUpdater struct {
	PollInterval time.Duration
	Workers      int
	mux          *p2p.PeerMuxer
	db           *leveldb.DB
	queue        *NameUpdateQueue
	nameLocker   util.MultiLocker
	bs           blob.Store
	obs          *util.Observable
	quitCh       chan struct{}
	wg           sync.WaitGroup
	lgr          log.Logger
}

func NewNameUpdater(mux *p2p.PeerMuxer, db *leveldb.DB, queue *NameUpdateQueue, nameLocker util.MultiLocker, bs blob.Store) *NameUpdater {
	return &NameUpdater{
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

func (u *NameUpdater) Start() error {
	for i := 0; i < u.Workers; i++ {
		u.wg.Add(1)
		go u.runWorker()
	}
	u.wg.Wait()
	return nil
}

func (u *NameUpdater) Stop() error {
	close(u.quitCh)
	u.wg.Wait()
	return nil
}

func (u *NameUpdater) OnUpdateProcessed(hdlr func(item *NameUpdateQueueItem, err error)) util.Unsubscriber {
	return u.obs.On("update:processed", hdlr)
}

func (u *NameUpdater) runWorker() {
	defer u.wg.Done()

	for {
		timer := time.NewTimer(u.PollInterval)
		select {
		case <-timer.C:
			item := u.queue.Dequeue()
			if item == nil {
				continue
			}

			cfg := &NameUpdateConfig{
				Mux:        u.mux,
				DB:         u.db,
				NameLocker: u.nameLocker,
				BlobStore:  u.bs,
				Item:       item,
			}
			if err := NameUpdateBlob(cfg); err != nil {
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

type NameUpdateConfig struct {
	Mux        *p2p.PeerMuxer
	DB         *leveldb.DB
	NameLocker util.MultiLocker
	BlobStore  blob.Store
	Item       *NameUpdateQueueItem
}

func NameUpdateBlob(cfg *NameUpdateConfig) error {
	item := cfg.Item
	defer item.Dispose()
	header, err := store.GetHeader(cfg.DB, item.Name)
	if err != nil && !errors.Is(err, leveldb.ErrNotFound) {
		return errors.Wrap(err, "error getting header")
	}

	info, err := store.GetNameInfo(cfg.DB, item.Name)
	if err != nil {
		return errors.Wrap(err, "error getting info")
	}

	var epochHeight, sectorSize uint16
	epochHeight = BlobEpoch(item.Name)
	if header != nil {
		epochHeight = header.EpochHeight
		sectorSize = header.SectorSize
	}

	subdomainMeta, err := NameSyncSubdomains(&NameSyncSubdomainsOpts{
		Timeout:     DefaultNameSyncerBlobResTimeout,
		Mux:         cfg.Mux,
		Peers:       item.PeerIDs,
		EpochHeight: epochHeight,
		SectorSize:  sectorSize,
		Name:        item.Name,
		DB:          cfg.DB,
	})

	if err != nil {
		return errors.Wrap(err, "error syncing subdomains")
	}

	for _, subdomain := range subdomainMeta.subdomains {
		name := fmt.Sprintf("%s.%s", subdomain.Name, item.Name)
		err = store.WithTx(cfg.DB, func(tx *leveldb.Transaction) error {
			if err := store.SetNameInfoTx(tx, name, subdomain.PublicKey, info.ImportHeight); err != nil {
				return errors.Wrap(err, "error inserting name info")
			}
			return nil
		})
	}

	err = store.WithTx(cfg.DB, func(tx *leveldb.Transaction) error {
		return store.SetSubdomainTx(tx, item.Name, subdomainMeta.subdomains)
	})

	if err != nil {
		return errors.Wrap(err, "error storing subdomain name info")
	}

	height, err := store.GetLastNameImportHeight(cfg.DB)
	if err != nil {
		nameUpdaterLogger.Error("error getting last name import height, skipping gossip", "err", err)
		return nil
	}
	if height-item.Height < 10 {
		nameUpdaterLogger.Info("updated name is below gossip height, skipping", "name", item.Name)
		return nil
	}

	update := &wire.NameUpdate{
		Name:          item.Name,
		EpochHeight:   item.EpochHeight,
		SubdomainSize: uint16(len(subdomainMeta.subdomains)),
	}
	p2p.GossipAll(cfg.Mux, update)
	return nil
}
