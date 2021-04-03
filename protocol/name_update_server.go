package protocol

import (
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/store"
	"fnd/util"
	"fnd/wire"

	"github.com/syndtr/goleveldb/leveldb"
)

type NameUpdateServer struct {
	mux        *p2p.PeerMuxer
	nameLocker util.MultiLocker
	db         *leveldb.DB
	lgr        log.Logger
}

func NewNameUpdateServer(mux *p2p.PeerMuxer, db *leveldb.DB, nameLocker util.MultiLocker) *NameUpdateServer {
	return &NameUpdateServer{
		mux:        mux,
		db:         db,
		nameLocker: nameLocker,
		lgr:        log.WithModule("update-server"),
	}
}

func (u *NameUpdateServer) Start() error {
	u.mux.AddMessageHandler(p2p.PeerMessageHandlerForType(wire.MessageTypeNameUpdateReq, u.UpdateReqHandler))
	return nil
}

func (u *NameUpdateServer) Stop() error {
	return nil
}

func (u *NameUpdateServer) UpdateReqHandler(peerID crypto.Hash, envelope *wire.Envelope) {
	msg := envelope.Message.(*wire.NameUpdateReq)
	u.lgr.Debug("receive update req", "name", msg.Name, "epoch", msg.EpochHeight, "sector", msg.SubdomainSize)

	if !u.nameLocker.TryRLock(msg.Name) {
		if err := u.mux.Send(peerID, wire.NewNameNilUpdate(msg.Name)); err != nil {
			u.lgr.Error("error sending response to update req", "name", msg.Name, "err", err)
		} else {
			u.lgr.Debug("serving nil update response for busy name", "name", msg.Name)
		}
		return
	}
	defer u.nameLocker.RUnlock(msg.Name)

	subdomains, err := store.GetSubdomains(u.db, msg.Name)
	if err != nil {
		if err := u.mux.Send(peerID, wire.NewNameNilUpdate(msg.Name)); err != nil {
			u.lgr.Error("error sending response to update req", "name", msg.Name, "err", err)
		} else {
			u.lgr.Debug("serving nil update response for future header", "name", msg.Name)
		}
		return
	}

	subdomainSize := uint16(len(subdomains))

	if subdomainSize < msg.SubdomainSize || subdomainSize == msg.SubdomainSize {
		if err := u.mux.Send(peerID, wire.NewNameNilUpdate(msg.Name)); err != nil {
			u.lgr.Error("error sending response to update req", "name", msg.Name, "err", err)
		} else {
			u.lgr.Debug("serving nil update response for future header", "name", msg.Name)
		}
		return
	}

	err = u.mux.Send(peerID, &wire.NameUpdate{
		Name:          msg.Name,
		SubdomainSize: subdomainSize,
	})
	if err != nil {
		u.lgr.Error("error serving update", "name", msg.Name, "err", err)
		return
	}

	u.lgr.Debug("served update", "name", msg.Name)
}
