package cmd

import (
	"fmt"
	"fnd/cli"
	"fnd/cmd/fnd-chat/cmd/chat"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fnd-chat",
	Short: "Command-line RPC interface for Footnote.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Int(cli.FlagRPCPort, 9098, "RPC port to connect to.")
	rootCmd.PersistentFlags().String(cli.FlagRPCHost, "127.0.0.1", "RPC host to connect to.")
	rootCmd.PersistentFlags().String(cli.FlagHome, "~/.fnd-cli", "Home directory for fnd's configuration.")
	rootCmd.PersistentFlags().String(cli.FlagFormat, "text", "Output format")
	chat.AddCmd(rootCmd)
}
