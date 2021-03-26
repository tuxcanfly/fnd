package blob

import (
	"context"
	"fnd/cli"
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
		publicKey := []byte(args[2])
		size, err := strconv.Atoi(args[3])
		if err != nil {
			return err
		}

		conn, err := cli.DialRPC(cmd)
		if err != nil {
			return err
		}

		client := apiv1.NewFootnotev1Client(conn)
		client.AddSubdomain(context.Background(), &apiv1.AddSubdomainReq{
			Name:      name,
			Subdomain: subdomain,
			PublicKey: publicKey,
			Size:      uint32(size),
		})
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&fndHome, "fnd-home", "~/.fnd", "Path to FootnoteD's home directory.")
	cmd.AddCommand(addCmd)
}
