package chat

import "github.com/spf13/cobra"

var cmd = &cobra.Command{
	Use:   "chat",
	Short: "Commands related to Footnote chat.",
}

func AddCmd(parent *cobra.Command) {
	parent.AddCommand(cmd)
}
