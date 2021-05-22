package protocol

import (
	"crypto/rand"
	"errors"
	"fnd/blob"
	"fnd/crypto"
	"fnd/store"
	"fnd/testutil/mockapp"
	"fnd/util"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

type updaterTestSetup struct {
	tp *mockapp.TestPeers
	ls *mockapp.TestStorage
	rs *mockapp.TestStorage
}

func TestNameUpdater(t *testing.T) {
	name := "bar"
	foo := &blob.Subdomain{
		ID:          0,
		Name:        "foo",
		EpochHeight: 10,
		Size:        128,
	}
	bar := &blob.Subdomain{
		ID:          0,
		Name:        "bar",
		EpochHeight: 10,
		Size:        128,
	}
	tests := []struct {
		name string
		run  func(t *testing.T, setup *updaterTestSetup)
	}{
		{
			"syncs subdomains when the local node has never seen the name",
			func(t *testing.T, setup *updaterTestSetup) {
				require.NoError(t, store.WithTx(setup.rs.DB, func(tx *leveldb.Transaction) error {
					sig, err := blob.NameSignSeal(setup.tp.RemoteSigner, foo)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "foo",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig,
						},
					}); err != nil {
						return err
					}
					return nil
				}))

				cfg := &NameUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &NameUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name: name,
						Pub:  setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, NameUpdateBlob(cfg))
				localSubdomains, err := store.GetSubdomains(setup.ls.DB, name)
				require.NoError(t, err)
				remoteSubdomains, err := store.GetSubdomains(setup.rs.DB, name)
				require.NoError(t, err)
				require.ElementsMatch(t, remoteSubdomains, localSubdomains)
			},
		},
		{
			"syncs subdomains when remote has a subdomain update",
			func(t *testing.T, setup *updaterTestSetup) {
				// local: ["bar.bar"]
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					sig, err := blob.NameSignSeal(setup.tp.RemoteSigner, bar)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "bar",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig,
						},
					}); err != nil {
						return err
					}
					return nil
				}))
				// remote: ["foo.bar", "bar.bar"]
				require.NoError(t, store.WithTx(setup.rs.DB, func(tx *leveldb.Transaction) error {
					sig1, err := blob.NameSignSeal(setup.tp.RemoteSigner, foo)
					if err != nil {
						return err
					}
					sig2, err := blob.NameSignSeal(setup.tp.RemoteSigner, bar)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "foo",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig1,
						},
						{
							Name:        "bar",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig2,
						},
					}); err != nil {
						return err
					}
					return nil
				}))

				cfg := &NameUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &NameUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name: name,
						Pub:  setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, NameUpdateBlob(cfg))
				localSubdomains, err := store.GetSubdomains(setup.ls.DB, name)
				require.NoError(t, err)
				remoteSubdomains, err := store.GetSubdomains(setup.rs.DB, name)
				require.NoError(t, err)
				require.ElementsMatch(t, remoteSubdomains, localSubdomains)
			},
		},
		{
			"aborts sync when there is an equivocation",
			func(t *testing.T, setup *updaterTestSetup) {
				// local: ["foo.bar"]
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					sig, err := blob.NameSignSeal(setup.tp.RemoteSigner, foo)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "foo",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig,
						},
					}); err != nil {
						return err
					}
					return nil
				}))
				// remote: ["bar.bar"]
				require.NoError(t, store.WithTx(setup.rs.DB, func(tx *leveldb.Transaction) error {
					sig, err := blob.NameSignSeal(setup.tp.RemoteSigner, bar)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "bar",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig,
						},
					}); err != nil {
						return err
					}
					return nil
				}))

				cfg := &NameUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &NameUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name: name,
						Pub:  setup.tp.RemoteSigner.Pub(),
					},
				}
				err := NameUpdateBlob(cfg)
				require.NotNil(t, err)
				time.Sleep(time.Second)
				require.True(t, errors.Is(err, ErrSubdomainEquivocation))
			},
		},
		{
			"aborts sync when there is a invalid payload signature",
			func(t *testing.T, setup *updaterTestSetup) {
				// use an invalid signature using local signer instead of remote
				require.NoError(t, store.WithTx(setup.rs.DB, func(tx *leveldb.Transaction) error {
					sig, err := blob.NameSignSeal(setup.tp.LocalSigner, foo)
					if err != nil {
						return err
					}
					if err := store.SetSubdomainTx(tx, name, []blob.Subdomain{
						{
							Name:        "foo",
							EpochHeight: 10,
							Size:        128,
							PublicKey:   setup.tp.RemoteSigner.Pub(),
							Signature:   sig,
						},
					}); err != nil {
						return err
					}
					return nil
				}))

				cfg := &NameUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &NameUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name: name,
						Pub:  setup.tp.RemoteSigner.Pub(),
					},
				}
				err := NameUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrInvalidSubdomainSignature))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPeers, peersDone := mockapp.ConnectTestPeers(t)
			defer peersDone()
			remoteStorage, remoteStorageDone := mockapp.CreateStorage(t)
			defer remoteStorageDone()
			localStorage, localStorageDone := mockapp.CreateStorage(t)
			defer localStorageDone()
			remoteSS := NewSubdomainServer(testPeers.RemoteMux, remoteStorage.DB, util.NewMultiLocker())
			require.NoError(t, remoteSS.Start())
			defer require.NoError(t, remoteSS.Stop())
			localSS := NewSubdomainServer(testPeers.LocalMux, localStorage.DB, util.NewMultiLocker())
			require.NoError(t, localSS.Start())
			defer require.NoError(t, localSS.Stop())
			go func() {
				remoteUQ := NewNameUpdateQueue(testPeers.RemoteMux, localStorage.DB)
				require.NoError(t, remoteUQ.Start())
				defer require.NoError(t, remoteUQ.Stop())
			}()

			require.NoError(t, store.WithTx(localStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetNameInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10); err != nil {
					return err
				}
				return nil
			}))

			require.NoError(t, store.WithTx(remoteStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetNameInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10); err != nil {
					return err
				}
				return nil
			}))

			tt.run(t, &updaterTestSetup{
				tp: testPeers,
				ls: localStorage,
				rs: remoteStorage,
			})
		})
	}
}

func TestBlobUpdater(t *testing.T) {
	name := "foo.bar"
	tests := []struct {
		name string
		run  func(t *testing.T, setup *updaterTestSetup)
	}{
		{
			"syncs sectors when the local node has never seen the blob",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				update := mockapp.FillBlobRandom(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					BlobEpoch(name),
					blob.SectorBytes,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
		{
			"syncs sectors when remote has a sector update",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(10)
				mockapp.FillBlobReader(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts.Add(-48*time.Hour),
					mockapp.LoremIpsumReader,
				)
				// Seek the reader to the start so that the update does not equivocate
				// i.e. it generates an update on top of the local blob
				_, err := mockapp.LoremIpsumReader.Seek(0, io.SeekStart)
				require.NoError(t, err)
				// create the new blob remotely
				update := mockapp.FillBlobReader(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize+10,
					ts,
					mockapp.LoremIpsumReader,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
		{
			"syncs sectors when remote has a epoch update (rollover)",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(10)
				mockapp.FillBlobReader(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts.Add(-48*time.Hour),
					mockapp.LoremIpsumReader,
				)
				// create the new blob remotely
				update := mockapp.FillBlobReader(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight+1,
					sectorSize,
					ts,
					mockapp.LoremIpsumReader,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
		{
			"aborts sync when there is an equivocation",
			func(t *testing.T, setup *updaterTestSetup) {
				require.NoError(t, store.WithTx(setup.rs.DB, func(tx *leveldb.Transaction) error {
					if err := store.SetInitialImportCompleteTx(tx); err != nil {
						return err
					}
					if err := store.SetNameInfoTx(tx, name, setup.tp.RemoteSigner.Pub(), 10); err != nil {
						return err
					}
					return nil
				}))
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(10)
				mockapp.FillBlobReader(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts.Add(-48*time.Hour),
					rand.Reader,
				)
				// create the new blob remotely
				// this forces an equivocation because local has 10 random sectors
				// and remote as 20 _different_ random sectors. Prev Hash at 10 will
				// mismatch, leading to ErrBlobUpdaterInvalidPrevHash
				update := mockapp.FillBlobReader(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize+10,
					ts,
					rand.Reader,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				time.Sleep(time.Second)
				require.True(t, errors.Is(err, ErrPayloadEquivocation))
			},
		},
		{
			"aborts sync when there is a invalid payload signature",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(10)
				mockapp.FillBlobReader(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts.Add(-48*time.Hour),
					mockapp.LoremIpsumReader,
				)
				// Use local signer on the remote (instead of remote signer
				// like other test cases above), so it generates an invalid
				// signature.
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					// TODO: ideally setup a fake signer
					if err := store.SetSubdomainInfoTx(tx, name, setup.tp.LocalSigner.Pub(), 10, blob.Size); err != nil {
						return err
					}
					return nil
				}))
				update := mockapp.FillBlobReader(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize+10,
					ts,
					mockapp.LoremIpsumReader,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrInvalidPayloadSignature))
			},
		},
		{
			"aborts sync if the new sector size is equal to the stored sector size",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(0)
				update := mockapp.FillBlobRandom(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: epochHeight,
						SectorSize:  sectorSize,
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterAlreadySynchronized))
			},
		},
		{
			"aborts sync if the name is locked",
			func(t *testing.T, setup *updaterTestSetup) {
				locker := util.NewMultiLocker()
				require.True(t, locker.TryLock(name))
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: locker,
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: BlobEpoch(name),
					},
				}
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterLocked))
			},
		},
		{
			"aborts sync if the new sector size is equal to the stored sector size",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				epochHeight := BlobEpoch(name)
				sectorSize := uint16(0)
				update := mockapp.FillBlobRandom(
					t,
					setup.ls.DB,
					setup.ls.BlobStore,
					setup.tp.RemoteSigner,
					name,
					epochHeight,
					sectorSize,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: epochHeight,
						SectorSize:  sectorSize,
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterAlreadySynchronized))
			},
		},
		{
			"aborts sync if the name is locked",
			func(t *testing.T, setup *updaterTestSetup) {
				locker := util.NewMultiLocker()
				require.True(t, locker.TryLock(name))
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: locker,
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: BlobEpoch(name),
					},
				}
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterLocked))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPeers, peersDone := mockapp.ConnectTestPeers(t)
			defer peersDone()
			remoteStorage, remoteStorageDone := mockapp.CreateStorage(t)
			defer remoteStorageDone()
			localStorage, localStorageDone := mockapp.CreateStorage(t)
			defer localStorageDone()
			remoteSS := NewSectorServer(testPeers.RemoteMux, remoteStorage.DB, remoteStorage.BlobStore, util.NewMultiLocker())
			require.NoError(t, remoteSS.Start())
			defer require.NoError(t, remoteSS.Stop())
			localSS := NewSectorServer(testPeers.LocalMux, localStorage.DB, localStorage.BlobStore, util.NewMultiLocker())
			require.NoError(t, localSS.Start())
			defer require.NoError(t, localSS.Stop())
			go func() {
				remoteUQ := NewBlobUpdateQueue(testPeers.RemoteMux, localStorage.DB)
				require.NoError(t, remoteUQ.Start())
				defer require.NoError(t, remoteUQ.Stop())
			}()

			require.NoError(t, store.WithTx(localStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetSubdomainInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10, blob.Size); err != nil {
					return err
				}
				return nil
			}))

			require.NoError(t, store.WithTx(remoteStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetSubdomainInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10, blob.Size); err != nil {
					return err
				}
				return nil
			}))

			tt.run(t, &updaterTestSetup{
				tp: testPeers,
				ls: localStorage,
				rs: remoteStorage,
			})
		})
	}
}

func TestEpoch(t *testing.T) {
	name := "foobar"
	tests := []struct {
		name string
		run  func(t *testing.T, setup *updaterTestSetup)
	}{
		{
			"syncs sectors when the local node has never seen the name before",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				update := mockapp.FillBlobRandom(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					0,
					blob.SectorBytes,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
		{
			"aborts sync if the name is banned",
			func(t *testing.T, setup *updaterTestSetup) {
				cfg := &BlobUpdateConfig{
					Mux:       setup.tp.LocalMux,
					DB:        setup.ls.DB,
					BlobStore: setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: BlobEpoch(name) + 1,
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					err := store.SetHeaderBan(tx, name, time.Time{})
					if err != nil {
						return err
					}
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: 0,
						SectorSize:  10,
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterBanned))
			},
		},
		{
			"syncs sectors when the name ban has passed",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				update := mockapp.FillBlobRandom(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					BlobEpoch(name)+1,
					blob.SectorBytes,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					err := store.SetHeaderBan(tx, name, time.Now().Add(-10*24*time.Duration(time.Hour)))
					if err != nil {
						return err
					}
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: 0,
						SectorSize:  0,
					}, blob.ZeroSectorHashes)
				}))
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
		{
			"aborts sync if the epoch is throttled",
			func(t *testing.T, setup *updaterTestSetup) {
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: BlobEpoch(name) - 1,
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:         name,
						EpochHeight:  0,
						SectorSize:   10,
						EpochStartAt: time.Now(),
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterInvalidEpochThrottled))
			},
		},
		{
			"aborts sync if the epoch is backdated",
			func(t *testing.T, setup *updaterTestSetup) {
				cfg := &BlobUpdateConfig{
					Mux:       setup.tp.LocalMux,
					DB:        setup.ls.DB,
					BlobStore: setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: 0,
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: BlobEpoch(name),
						SectorSize:  10,
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterInvalidEpochBackdated))
			},
		},
		{
			"aborts sync if the epoch is futuredated",
			func(t *testing.T, setup *updaterTestSetup) {
				cfg := &BlobUpdateConfig{
					Mux:       setup.tp.LocalMux,
					DB:        setup.ls.DB,
					BlobStore: setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: BlobEpoch(name) + 2,
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: BlobEpoch(name),
						SectorSize:  10,
					}, blob.ZeroSectorHashes)
				}))
				err := BlobUpdateBlob(cfg)
				require.NotNil(t, err)
				require.True(t, errors.Is(err, ErrBlobUpdaterInvalidEpochFuturedated))
			},
		},
		{
			"rewrites partial blob with new blob on epoch rollover",
			func(t *testing.T, setup *updaterTestSetup) {
				ts := time.Now()
				update := mockapp.FillBlobRandom(
					t,
					setup.rs.DB,
					setup.rs.BlobStore,
					setup.tp.RemoteSigner,
					name,
					BlobEpoch(name),
					blob.SectorBytes,
					ts,
				)
				cfg := &BlobUpdateConfig{
					Mux:        setup.tp.LocalMux,
					DB:         setup.ls.DB,
					NameLocker: util.NewMultiLocker(),
					BlobStore:  setup.ls.BlobStore,
					Item: &BlobUpdateQueueItem{
						PeerIDs: NewPeerSet([]crypto.Hash{
							crypto.HashPub(setup.tp.RemoteSigner.Pub()),
						}),
						Name:        name,
						EpochHeight: update.EpochHeight,
						SectorSize:  update.SectorSize,
						Pub:         setup.tp.RemoteSigner.Pub(),
					},
				}
				require.NoError(t, store.WithTx(setup.ls.DB, func(tx *leveldb.Transaction) error {
					return store.SetHeaderTx(tx, &store.Header{
						Name:        name,
						EpochHeight: BlobEpoch(name) - 1,
						SectorSize:  10,
					}, blob.ZeroSectorHashes)
				}))
				require.NoError(t, BlobUpdateBlob(cfg))
				mockapp.RequireBlobsEqual(t, setup.ls.BlobStore, setup.rs.BlobStore, name)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPeers, peersDone := mockapp.ConnectTestPeers(t)
			defer peersDone()
			remoteStorage, remoteStorageDone := mockapp.CreateStorage(t)
			defer remoteStorageDone()
			localStorage, localStorageDone := mockapp.CreateStorage(t)
			defer localStorageDone()
			remoteSS := NewSectorServer(testPeers.RemoteMux, remoteStorage.DB, remoteStorage.BlobStore, util.NewMultiLocker())
			require.NoError(t, remoteSS.Start())
			defer require.NoError(t, remoteSS.Stop())

			require.NoError(t, store.WithTx(localStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetSubdomainInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10, blob.Size); err != nil {
					return err
				}
				return nil
			}))

			require.NoError(t, store.WithTx(remoteStorage.DB, func(tx *leveldb.Transaction) error {
				if err := store.SetInitialImportCompleteTx(tx); err != nil {
					return err
				}
				if err := store.SetSubdomainInfoTx(tx, name, testPeers.RemoteSigner.Pub(), 10, blob.Size); err != nil {
					return err
				}
				return nil
			}))

			tt.run(t, &updaterTestSetup{
				tp: testPeers,
				ls: localStorage,
				rs: remoteStorage,
			})
		})
	}
}
