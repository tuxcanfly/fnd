package name

import "github.com/spf13/cobra"

var cmd = &cobra.Command{
	Use:   "name",
	Short: "Commands related to Footnote name.",
}

func AddCmd(parent *cobra.Command) {
	parent.AddCommand(cmd)
}
