package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"fnd/blob"
	"fnd/cli"
	"fnd/config"
	"fnd/crypto"
	"fnd/log"
	"fnd/p2p"
	"fnd/protocol"
	"fnd/rpc"
	"fnd/service"
	"fnd/store"
	"fnd/util"
	"fnd/version"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	apiv1 "fnd/rpc/v1"

	"fnd.localhost/handshake/client"
	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/grpc"
)

var (
	initialize        bool
	name              string
	connection        *grpc.ClientConn
	configuredHomeDir string
	pubkey            string
	lgr               log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "fnd",
	Short: "Footnote Daemon",
	RunE: func(rootCmd *cobra.Command, args []string) error {
		configuredHomeDir = cli.GetHomeDir(rootCmd)
		cfg, err := config.ReadConfigFile(configuredHomeDir)
		var perr *os.PathError
		if errors.As(err, &perr) && os.IsNotExist(perr) {
			initialize = true
			if err := config.InitHomeDir(configuredHomeDir); err != nil {
				return errors.Wrap(err, "error ensuring home directory")
			}
			cfg, err = config.ReadConfigFile(configuredHomeDir)
			if err != nil {
				return errors.Wrap(err, "error reading config file")
			}
		} else if err != nil {
			return errors.Wrap(err, "error reading config file")
		}
		signer, err := cli.GetSigner(configuredHomeDir)
		if err != nil {
			return errors.Wrap(err, "error opening home directory")
		}

		pubkey = base64.StdEncoding.EncodeToString(signer.Pub().SerializeCompressed())

		rpcHost := cfg.RPC.Host
		rpcPort := cfg.RPC.Port
		p2pHost := cfg.P2P.Host
		logLevel, err := log.NewLevel(cfg.LogLevel)
		if err != nil {
			return errors.Wrap(err, "error parsing log level")
		}
		log.SetLevel(logLevel)
		log.SetOutput(path.Join(configuredHomeDir, "debug.log"))
		lgr = log.WithModule("main")

		lgr.Info("starting fnd", "git_commit", version.GitCommit, "git_tag", version.GitTag)
		lgr.Info("opening home directory", "path", configuredHomeDir)

		dbPath := config.ExpandDBPath(configuredHomeDir)
		lgr.Info("opening db", "path", dbPath)
		db, err := store.Open(dbPath)
		if err != nil {
			return err
		}

		blobsPath := config.ExpandBlobsPath(configuredHomeDir)
		lgr.Info("opening blob store", "path", blobsPath)
		bs := blob.NewStore(blobsPath)

		seedsStr := cfg.P2P.FixedSeeds
		seeds, err := p2p.ParseSeedPeers(seedsStr)
		if err != nil {
			return errors.Wrap(err, "error parsing seed peers")
		}

		var dnsSeeds []string
		if len(cfg.P2P.DNSSeeds) != 0 {
			seenSeeds := make(map[string]bool)
			for _, domain := range cfg.P2P.DNSSeeds {
				lgr.Info("looking up DNS seeds", "domain", domain)
				seeds, err := p2p.ResolveDNSSeeds(domain)
				if err != nil {
					lgr.Error("error resolving DNS seeds", "domain", domain)
					continue
				}
				for _, seed := range seeds {
					if seenSeeds[seed] == true {
						continue
					}
					seenSeeds[seed] = true
					dnsSeeds = append(dnsSeeds, seed)
				}
			}
		}

		lgr.Info("ingesting ban lists")
		if len(cfg.BanLists) > 0 {
			if err := protocol.IngestBanLists(db, bs, cfg.BanLists); err != nil {
				return errors.Wrap(err, "failed to ingest ban lists")
			}
		}

		var services []service.Service
		mux := p2p.NewPeerMuxer(p2p.MainnetMagic, signer)
		pmCfg := &p2p.PeerManagerOpts{
			Mux:         mux,
			DB:          db,
			SeedPeers:   seeds,
			Signer:      signer,
			ListenHost:  p2pHost,
			MaxInbound:  cfg.P2P.MaxInboundPeers,
			MaxOutbound: cfg.P2P.MaxOutboundPeers,
		}
		pm := p2p.NewPeerManager(pmCfg)
		services = append(services, pm)

		if p2pHost != "" && p2pHost != "127.0.0.1" {
			services = append(services, p2p.NewListener(p2pHost, pm))
		}
		c := client.NewClient(
			cfg.HNSResolver.Host,
			client.WithAPIKey(cfg.HNSResolver.APIKey),
			client.WithPort(cfg.HNSResolver.Port),
			client.WithBasePath(cfg.HNSResolver.BasePath),
		)

		lgr.Info("connecting to HSD", "host", cfg.HNSResolver.Host)
		maxHSDRetries := 10
		for i := 0; i < maxHSDRetries; maxHSDRetries++ {
			if _, err := c.GetInfo(); err != nil {
				lgr.Warn("error connecting to HSD, retrying in 10 seconds", "err", err)
				if i == maxHSDRetries-1 {
					return fmt.Errorf("could not connect to HSD after %d retries", maxHSDRetries)
				}
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}

		nameLocker := util.NewMultiLocker()
		ownPeerID := crypto.HashPub(signer.Pub())

		importer := protocol.NewNameImporter(c, db)
		importer.ConfirmationDepth = cfg.Tuning.NameImporter.ConfirmationDepth
		importer.CheckInterval = config.ConvertDuration(cfg.Tuning.NameImporter.CheckIntervalMS, time.Millisecond)
		importer.Workers = cfg.Tuning.NameImporter.Workers
		importer.VerificationThreshold = cfg.Tuning.NameImporter.VerificationThreshold

		updateQueue := protocol.NewUpdateQueue(mux, db)
		updateQueue.MaxLen = int32(cfg.Tuning.UpdateQueue.MaxLen)

		updater := protocol.NewUpdater(mux, db, updateQueue, nameLocker, bs)
		updater.PollInterval = config.ConvertDuration(cfg.Tuning.Updater.PollIntervalMS, time.Millisecond)
		updater.Workers = cfg.Tuning.Updater.Workers

		pinger := protocol.NewPinger(mux)

		sectorServer := protocol.NewSectorServer(mux, db, bs, nameLocker)
		sectorServer.CacheExpiry = config.ConvertDuration(cfg.Tuning.SectorServer.CacheExpiryMS, time.Millisecond)

		updateServer := protocol.NewUpdateServer(mux, db, nameLocker)

		peerExchanger := protocol.NewPeerExchanger(pm, mux, db)
		peerExchanger.SampleSize = cfg.Tuning.PeerExchanger.SampleSize
		peerExchanger.ResponseTimeout = config.ConvertDuration(cfg.Tuning.PeerExchanger.ResponseTimeoutMS, time.Millisecond)
		peerExchanger.RequestInterval = config.ConvertDuration(cfg.Tuning.PeerExchanger.RequestIntervalMS, time.Millisecond)

		nameSyncer := protocol.NewNameSyncer(mux, db, nameLocker, updater)
		nameSyncer.Workers = cfg.Tuning.NameSyncer.Workers
		nameSyncer.SampleSize = cfg.Tuning.NameSyncer.SampleSize
		nameSyncer.UpdateResponseTimeout = config.ConvertDuration(cfg.Tuning.NameSyncer.UpdateResponseTimeoutMS, time.Millisecond)
		nameSyncer.Interval = config.ConvertDuration(cfg.Tuning.NameSyncer.IntervalMS, time.Millisecond)
		nameSyncer.SyncResponseTimeout = config.ConvertDuration(cfg.Tuning.NameSyncer.SyncResponseTimeoutMS, time.Millisecond)

		server := rpc.NewServer(&rpc.Opts{
			PeerID:      ownPeerID,
			Mux:         mux,
			DB:          db,
			BlobStore:   bs,
			PeerManager: pm,
			NameLocker:  nameLocker,
			Host:        rpcHost,
			Port:        rpcPort,
		})
		services = append(services, []service.Service{
			importer,
			updateQueue,
			updater,
			pinger,
			sectorServer,
			updateServer,
			peerExchanger,
			nameSyncer,
			server,
		}...)

		if cfg.Heartbeat.URL != "" {
			hb := protocol.NewHeartbeater(cfg.Heartbeat.URL, cfg.Heartbeat.Moniker, ownPeerID)
			services = append(services, hb)
		}

		lgr.Info("starting services")
		for _, s := range services {
			go func(s service.Service) {
				if err := s.Start(); err != nil {
					lgr.Error("failed to start service", "err", err)
				}
			}(s)
		}

		if cfg.EnableProfiler {
			lgr.Info("starting profiler", "port", 6060)
			runtime.SetBlockProfileRate(1)
			runtime.SetMutexProfileFraction(1)
			go func() {
				err := http.ListenAndServe("localhost:6060", nil)
				lgr.Error("error starting profiler", "err", err)
			}()
		}

		err = store.WithTx(db, func(tx *leveldb.Transaction) error {
			for _, seed := range seeds {
				if err := store.WhitelistPeerTx(tx, seed.IP); err != nil {
					return err
				}
			}
			for _, seed := range dnsSeeds {
				if err := store.WhitelistPeerTx(tx, seed); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "error whitelisting seed peers")
		}

		lgr.Info("dialing seed peers")
		for _, seed := range seeds {
			if err := pm.DialPeer(seed.ID, seed.IP, true); err != nil {
				lgr.Warn("error dialing seed peer", "err", err)
				continue
			}
		}
		for _, seed := range dnsSeeds {
			if err := pm.DialPeer(crypto.ZeroHash, seed, false); err != nil {
				lgr.Warn("error dialing DNS seed peer", "err", err)
			}
		}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		connection, err = cli.DialRPC()
		if err != nil {
			lgr.Fatal(err.Error())
		}

		g, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			lgr.Fatal(err.Error())
		}
		defer g.Close()

		g.SetManagerFunc(Layout)
		g.SetKeybinding("name", gocui.KeyEnter, gocui.ModNone, Connect)
		g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, Send)
		g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, Disconnect)
		if !initialize {
			g.Update(sync)
		}
		g.MainLoop()

		sig := <-sigs
		lgr.Info("shutting down", "signal", sig)
		return nil
	},
}

// Layout creates chat UI
func Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	g.Cursor = true

	if messages, err := g.SetView("messages", 0, 0, maxX-20, maxY-5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		messages.Title = " messages: "
		messages.Autoscroll = true
		messages.Wrap = true
	}

	if identity, err := g.SetView("identity", 0, maxY-10, maxX-20, maxY-8); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		identity.Title = " identity: "
		identity.Autoscroll = false
		identity.Wrap = true
		identity.Editable = false
	}

	if input, err := g.SetView("input", 0, maxY-5, maxX-20, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		input.Title = fmt.Sprintf(" [ %s ] send: ", name)
		input.Autoscroll = false
		input.Wrap = true
		input.Editable = true
	}

	if users, err := g.SetView("users", maxX-20, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		users.Title = " users: "
		users.Autoscroll = false
		users.Wrap = true
	}

	if initialize {
		if name, err := g.SetView("name", maxX/2-10, maxY/2-1, maxX/2+10, maxY/2+1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			g.SetCurrentView("name")
			name.Title = " name: "
			name.Autoscroll = false
			name.Wrap = true
			name.Editable = true
		}
	} else {
		g.SetCurrentView("input")
	}
	return nil
}

// Disconnect from chat and close
func Disconnect(g *gocui.Gui, v *gocui.View) error {
	connection.Close()
	return gocui.ErrQuit
}

// Send message
func Send(g *gocui.Gui, v *gocui.View) error {
	signer, err := cli.GetSigner(configuredHomeDir)
	if err != nil {
		lgr.Fatal(err.Error())
	}

	lgr.Debug("name", name)
	writer := rpc.NewBlobWriter(apiv1.NewFootnotev1Client(connection), signer, name)

	if err := writer.Open(); err != nil {
		lgr.Fatal(err.Error())
	}

	ts := []byte("\n" + strconv.FormatInt(time.Now().UTC().Unix(), 10) + ": ")
	msg := append(ts, []byte(v.Buffer()+"\n")...)
	rd := bufio.NewReader(bytes.NewReader(msg))
	var sector blob.Sector
	for i := 0; i < blob.MaxSectors; i++ {
		if _, err := rd.Read(sector[:]); err != nil {
			if err == io.EOF {
				break
			}
			lgr.Fatal(err.Error())
		}
		writer.WriteSector(sector[:])
	}
	_, err = writer.Commit(true)
	if err != nil {
		lgr.Fatal(err.Error())
	}

	g.Update(func(g *gocui.Gui) error {
		v.Clear()
		v.SetCursor(0, 0)
		v.SetOrigin(0, 0)
		return nil
	})
	return nil
}

// Connect to the server, create new reader, writer and set client name
func Connect(g *gocui.Gui, v *gocui.View) error {
	// Some UI changes
	g.SetViewOnTop("messages")
	g.SetViewOnTop("users")
	g.SetViewOnTop("input")
	g.SetViewOnTop("identity")
	g.SetCurrentView("input")

	name = strings.TrimSpace(v.Buffer())
	err := config.WriteName(configuredHomeDir, name)
	lgr.Error(err.Error())

	sync(g)
	return nil
}

func sync(g *gocui.Gui) error {
	var err error
	name, err = config.ReadName(configuredHomeDir)

	if err != nil {
		return err
	}
	identityView, _ := g.View("identity")
	fmt.Fprintln(identityView, pubkey)

	inputView, _ := g.View("input")
	inputView.Title = fmt.Sprintf(" [ %s ] send: ", name)

	go func() {
		for {
			time.Sleep(10 * time.Second)
			// Wait for server messages in new goroutine
			messagesView, _ := g.View("messages")
			usersView, _ := g.View("users")

			clientsCount := 0
			var clients string
			messagesView.Clear()
			err := rpc.ListBlobInfo(apiv1.NewFootnotev1Client(connection), "", func(info *store.BlobInfo) bool {
				clients += info.Name + "\n"
				clientsCount++

				reader := rpc.NewBlobReader(apiv1.NewFootnotev1Client(connection), info.Name)
				buffer := make([]byte, blob.SectorBytes*info.SectorSize)
				_, err := reader.Read(buffer)
				if err != nil && err != io.EOF {
					lgr.Fatal(err.Error())
				}
				msg := string(buffer)
				g.Update(func(g *gocui.Gui) error {
					fmt.Fprintln(messagesView, info.Name+": ")
					fmt.Fprintln(messagesView, strings.TrimSpace(msg))
					return nil
				})

				return true
			})
			if err != nil {
				lgr.Fatal(err.Error())
			}

			g.Update(func(g *gocui.Gui) error {
				usersView.Title = fmt.Sprintf(" %d users: ", clientsCount)
				usersView.Clear()
				fmt.Fprintln(usersView, clients)
				return nil
			})
		}
	}()
	return nil
}

func init() {
	rootCmd.PersistentFlags().String(cli.FlagHome, "~/.fnd", "Home directory for the daemon's config and database.")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
