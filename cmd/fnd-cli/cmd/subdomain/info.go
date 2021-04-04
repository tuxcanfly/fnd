package subdomain

import (
	"encoding/hex"
	"fmt"
	"fnd/cli"
	"fnd/rpc"
	apiv1 "fnd/rpc/v1"
	"os"
	"strings"

	"fnd.localhost/handshake/primitives"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <names>",
	Short: "Returns metadata about Footnote blobs.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		names := strings.Split(args[0], ",")
		for _, name := range names {
			if err := primitives.ValidateName(name); err != nil {
				return errors.Wrap(err, fmt.Sprintf("invalid name %s", name))
			}
		}

		conn, err := cli.DialRPC(cmd)
		if err != nil {
			return err
		}
		grpcClient := apiv1.NewFootnotev1Client(conn)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{
			"Name",
			"Public Key",
			"Import Height",
		})

		for _, name := range names {
			res, err := rpc.GetNameInfo(grpcClient, name)
			if err != nil {
				return err
			}

			table.Append([]string{
				res.Name,
				hex.EncodeToString(res.PublicKey.SerializeCompressed()),
				string(res.ImportHeight),
			})
		}

		table.Render()
		return nil
	},
}

func init() {
	cmd.AddCommand(infoCmd)
}
