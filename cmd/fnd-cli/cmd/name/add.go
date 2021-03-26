package name

import (
	"context"
	"fmt"
	"fnd/cli"
	"fnd/protocol"
	apiv1 "fnd/rpc/v1"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	fndHome string
)

var addCmd = &cobra.Command{
	Use:   "add <name> <subdomain> <publickey> <size>",
	Short: "Add subdomain to the given name.",
	Args:  cobra.MinimumNArgs(1),
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

		// TODO: signer, must sign subdomain data with domain pubkey

		client := apiv1.NewFootnotev1Client(conn)
		_, err = client.AddSubdomain(context.Background(), &apiv1.AddSubdomainReq{
			Name:      name,
			Subdomain: subdomain,
			PublicKey: publicKey.SerializeCompressed(),
			Size:      uint32(size),
		})
		fmt.Println(err)
		return err
	},
}

func init() {
	addCmd.Flags().StringVar(&fndHome, "fnd-home", "~/.fnd", "Path to FootnoteD's home directory.")
	cmd.AddCommand(addCmd)
}
