package name

import (
	"context"
	"fnd/blob"
	"fnd/cli"
	"fnd/protocol"
	apiv1 "fnd/rpc/v1"
	"strconv"

	"github.com/spf13/cobra"
)

const (
	BroadcastFlag = "broadcast"
)

var (
	fndHome   string
	broadcast bool
)

var addCmd = &cobra.Command{
	Use:   "add <name> <subdomain> <publickey> <size>",
	Short: "Add subdomain to the given name.",
	Args:  cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		subdomain := args[1]
		publicKeyStr := args[2]
		size, err := strconv.Atoi(args[3])
		if err != nil {
			return err
		}

		conn, err := cli.DialRPC(cmd)
		if err != nil {
			return err
		}

		publicKey, err := protocol.ParseFNRecord(publicKeyStr)
		if err != nil {
			return err
		}

		homeDir := cli.GetHomeDir(cmd)
		signer, err := cli.GetSigner(homeDir)
		if err != nil {
			return err
		}

		epochHeight := protocol.NameEpoch(subdomain)

		sig, err := blob.NameSignSeal(signer, subdomain, epochHeight, uint8(size))
		if err != nil {
			return err
		}

		client := apiv1.NewFootnotev1Client(conn)
		_, err = client.AddSubdomain(context.Background(), &apiv1.AddSubdomainReq{
			Name:        name,
			Subdomain:   subdomain,
			EpochHeight: uint32(epochHeight),
			Size:        uint32(size),
			PublicKey:   publicKey.SerializeCompressed(),
			Signature:   sig[:],
			Broadcast:   broadcast,
		})
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&broadcast, BroadcastFlag, true, "Broadcast data to the network upon completion")
	addCmd.Flags().StringVar(&fndHome, "fnd-home", "~/.fnd", "Path to FootnoteD's home directory.")
	cmd.AddCommand(addCmd)
}
