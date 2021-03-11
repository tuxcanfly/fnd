package chat

import (
	"bufio"
	"bytes"
	"fmt"
	"fnd/blob"
	"fnd/cli"
	"fnd/rpc"
	apiv1 "fnd/rpc/v1"
	"fnd/store"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var fndHome string

var (
	name       string
	connection *grpc.ClientConn
)

var chatCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a demo chat app.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		connection, err = cli.DialRPC(cmd)
		if err != nil {
			log.Fatal(err)
		}

		g, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			log.Fatal(err)
		}
		defer g.Close()

		g.SetManagerFunc(Layout)
		g.SetKeybinding("name", gocui.KeyEnter, gocui.ModNone, Connect)
		g.SetKeybinding("input", gocui.KeyEnter, gocui.ModNone, Send)
		g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, Disconnect)
		g.MainLoop()
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

	if input, err := g.SetView("input", 0, maxY-5, maxX-20, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		input.Title = " send: "
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
	return nil
}

// Disconnect from chat and close
func Disconnect(g *gocui.Gui, v *gocui.View) error {
	connection.Close()
	return gocui.ErrQuit
}

// Send message
func Send(g *gocui.Gui, v *gocui.View) error {
	homeDir := cli.GetHomeDir(cmd)
	signer, err := cli.GetSigner(homeDir)
	if err != nil {
		log.Fatal(err)
	}

	writer := rpc.NewBlobWriter(apiv1.NewFootnotev1Client(connection), signer, name)

	if err := writer.Open(); err != nil {
		log.Fatal(err)
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
			log.Fatal(err)
		}
		writer.WriteSector(sector[:])
	}
	_, err = writer.Commit(true)
	if err != nil {
		log.Fatal(err)
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
	g.SetCurrentView("input")

	name = strings.TrimSpace(v.Buffer())

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
					log.Fatal(err)
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
				log.Fatal(err)
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
	chatCmd.Flags().StringVar(&fndHome, "fnd-home", "~/.fnd-cli", "Path to fnd's home directory.")
	cmd.AddCommand(chatCmd)
}
