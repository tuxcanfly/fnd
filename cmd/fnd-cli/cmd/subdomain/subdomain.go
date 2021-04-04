package subdomain

import "github.com/spf13/cobra"

var cmd = &cobra.Command{
	Use:   "subdomain",
	Short: "Commands related to Footnote subdomains.",
}

func AddCmd(parent *cobra.Command) {
	parent.AddCommand(cmd)
}
