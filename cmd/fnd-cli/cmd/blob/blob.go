package blob

import "github.com/spf13/cobra"

var cmd = &cobra.Command{
	Use:   "blob",
	Short: "Commands related to Footnote blobs.",
}

func AddCmd(parent *cobra.Command) {
	parent.AddCommand(cmd)
}
