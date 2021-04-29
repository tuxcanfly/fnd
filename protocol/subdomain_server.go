package protocol

import (
	"bytes"
	"fnd/blob"
	"fnd/config"
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/util"
	"fnd/wire"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

type SubdomainServer struct {
	CacheExpiry time.Duration
	mux         *p2p.PeerMuxer
	db          *leveldb.DB
	nameLocker  util.MultiLocker
	lgr         log.Logger
	cache       *util.Cache
}

func NewSubdomainServer(mux *p2p.PeerMuxer, db *leveldb.DB, nameLocker util.MultiLocker) *SubdomainServer {
	return &SubdomainServer{
		CacheExpiry: config.ConvertDuration(config.DefaultConfig.Tuning.SectorServer.CacheExpiryMS, time.Millisecond),
		mux:         mux,
		db:          db,
		nameLocker:  nameLocker,
		cache:       util.NewCache(),
		lgr:         log.WithModule("sector-server"),
	}
}

func (s *SubdomainServer) Start() error {
	s.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeNameReq, s.onNameReq))
	s.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeNameEquivocationProof, s.onEquivocationProof))
	return nil
}

func (s *SubdomainServer) Stop() error {
	return nil
}

// onNameReq handles a NameReq and responds with a NameRes
//
// In case the NameReq contains SubdomainSize == MaxSubdomains, it is considered a
// special case, which is a part of the equivocation proof flow. Note that in a
// normal NameReq/NameRes flow, SubdomainSize may never be MaxSubdomains as it is the
// start for the expected payload.
//
// In the special case where SubdomainSize = MaxSubdomains, the NameReq is considered
// an Equivocation Request. In response, the peer is sent an Equivocation Proof.
//
// Equivocation Proof flow:
// Syncer - sees a conflicting update
// - Writes equivocation proof locally to db
// - Sends an equivocation Update (special case where Update.SubdomainSize = 0)
// - Expects peers to reply with Equivocation Request i.e. NameReq where SubdomainSize = MaxSubdomains
// - Responds with Equivocation Proof
func (s *SubdomainServer) onNameReq(peerID crypto.Hash, envelope *wire.Envelope) {
	reqMsg := envelope.Message.(*wire.NameReq)
	lgr := s.lgr.Sub(
		"name", reqMsg.Name,
		"peer_id", peerID,
	)

	// Name request with SubdomainSize == blob.MaxSubdomains is an equivocation
	// request. Respond with equivocation proof.
	if reqMsg.SubdomainSize == blob.MaxSubdomains {
		raw, err := store.GetNameEquivocationProof(s.db, reqMsg.Name)
		if err != nil {
			lgr.Error(
				"failed to fetch equivocation proof",
				"err", err)
			return
		}
		proof := &wire.NameEquivocationProof{}
		buf := bytes.NewReader(raw)
		if err := proof.Decode(buf); err != nil {
			lgr.Error(
				"failed to deserialize equivocation proof",
				"err", err)
			return
		}
		if err := s.mux.Send(peerID, proof); err != nil {
			s.lgr.Error("error serving equivocation proof", "err", err)
			return
		}
		return
	}

	if !s.nameLocker.TryRLock(reqMsg.Name) {
		lgr.Info("dropping sector req for busy name")
		return
	}

	cacheKey := reqMsg.Name
	cached := s.cache.Get(cacheKey)
	if cached != nil {
		s.nameLocker.RUnlock(reqMsg.Name)
		s.sendResponse(peerID, reqMsg.Name, cached.([]blob.Subdomain))
		return
	}

	subdomains, err := store.GetSubdomains(s.db, reqMsg.Name)
	if err != nil {
		lgr.Error(
			"failed to fetch subdomains",
			"err", err)
		return
	}
	s.cache.Set(cacheKey, subdomains, int64(s.CacheExpiry/time.Millisecond))
	s.nameLocker.RUnlock(reqMsg.Name)
	s.sendResponse(peerID, reqMsg.Name, subdomains)
}

func (s *SubdomainServer) sendResponse(peerID crypto.Hash, name string, subdomains []blob.Subdomain) {
	resMsg := &wire.NameRes{
		Name:       name,
		Subdomains: subdomains,
	}
	if err := s.mux.Send(peerID, resMsg); err != nil {
		s.lgr.Error("error serving subdomain response", "err", err)
		return
	}
	s.lgr.Debug(
		"served subdomain response",
		"peer_id", peerID,
	)
}

func (s *SubdomainServer) onEquivocationProof(peerID crypto.Hash, envelope *wire.Envelope) {
	return
}
