package protocol

import (
	"fmt"
	"time"

	"github.com/ddrp-org/ddrp/blob"
	"github.com/ddrp-org/ddrp/config"
	"github.com/ddrp-org/ddrp/crypto"
	"github.com/ddrp-org/ddrp/log"
	"github.com/ddrp-org/ddrp/p2p"
	"github.com/ddrp-org/ddrp/store"
	"github.com/ddrp-org/ddrp/util"
	"github.com/ddrp-org/ddrp/wire"
	"github.com/syndtr/goleveldb/leveldb"
)

type SectorServer struct {
	CacheExpiry time.Duration
	mux         *p2p.PeerMuxer
	db          *leveldb.DB
	bs          blob.Store
	nameLocker  util.MultiLocker
	lgr         log.Logger
	cache       *util.Cache
}

func NewSectorServer(mux *p2p.PeerMuxer, db *leveldb.DB, bs blob.Store, nameLocker util.MultiLocker) *SectorServer {
	return &SectorServer{
		CacheExpiry: config.ConvertDuration(config.DefaultConfig.Tuning.SectorServer.CacheExpiryMS, time.Millisecond),
		mux:         mux,
		db:          db,
		bs:          bs,
		nameLocker:  nameLocker,
		cache:       util.NewCache(),
		lgr:         log.WithModule("sector-server"),
	}
}

func (s *SectorServer) Start() error {
	s.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeBlobReq, s.onBlobReq))
	return nil
}

func (s *SectorServer) Stop() error {
	return nil
}

func (s *SectorServer) onBlobReq(peerID crypto.Hash, envelope *wire.Envelope) {
	reqMsg := envelope.Message.(*wire.BlobReq)
	lgr := s.lgr.Sub(
		"name", reqMsg.Name,
		"peer_id", peerID,
	)

	if !s.nameLocker.TryRLock(reqMsg.Name) {
		lgr.Info("dropping sector req for busy name")
		return
	}

	header, err := store.GetHeader(s.db, reqMsg.Name)
	if err != nil {
		lgr.Error(
			"failed to fetch header",
			"err", err)
		return
	}

	// FIXME: skip if sectorSize == 0; we know it will be ZeroHash
	prevHash, err := store.GetSectorHash(s.db, reqMsg.Name, reqMsg.SectorSize-1)
	if err != nil {
		lgr.Error(
			"failed to fetch sector hash",
			"err", err)
		return
	}

	cacheKey := fmt.Sprintf("%s:%d:%d", reqMsg.Name, reqMsg.EpochHeight, reqMsg.SectorSize)
	cached := s.cache.Get(cacheKey)
	if cached != nil {
		s.nameLocker.RUnlock(reqMsg.Name)
		s.sendResponse(peerID, reqMsg.Name, prevHash, cached.([]blob.Sector), reqMsg.EpochHeight, reqMsg.SectorSize)
		return
	}

	bl, err := s.bs.Open(reqMsg.Name)
	if err != nil {
		s.nameLocker.RUnlock(reqMsg.Name)
		lgr.Error(
			"failed to fetch blob",
			"err", err,
		)
		return
	}
	defer func() {
		if err := bl.Close(); err != nil {
			s.lgr.Error("failed to close blob", "err", err)
		}
	}()
	var sectors []blob.Sector
	for i := reqMsg.SectorSize; i < header.SectorSize; i++ {
		sector := &blob.Sector{}
		_, err = bl.ReadAt(sector[:], int64(i)*blob.SectorLen)
		if err != nil {
			s.nameLocker.RUnlock(reqMsg.Name)
			lgr.Error(
				"failed to read sector",
				"err", err,
			)
			return
		}
		sectors = append(sectors, *sector)
	}
	s.cache.Set(cacheKey, sectors, int64(s.CacheExpiry/time.Millisecond))
	s.nameLocker.RUnlock(reqMsg.Name)
	s.sendResponse(peerID, reqMsg.Name, prevHash, sectors, reqMsg.EpochHeight, reqMsg.SectorSize)
}

func (s *SectorServer) sendResponse(peerID crypto.Hash, name string, prevHash crypto.Hash, sectors []blob.Sector, epochHeight, sectorSize uint16) {
	resMsg := &wire.BlobRes{
		Name:            name,
		EpochHeight:     epochHeight,
		PayloadPosition: sectorSize,
		PrevHash:        prevHash,
		Payload:         sectors,
	}
	if err := s.mux.Send(peerID, resMsg); err != nil {
		s.lgr.Error("error serving sector response", "err", err)
		return
	}
	s.lgr.Debug(
		"served sector response",
		"peer_id", peerID,
		"sector_size", sectorSize,
	)
}
