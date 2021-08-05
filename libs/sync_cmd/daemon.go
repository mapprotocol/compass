package sync_cmd

import (
	"github.com/spf13/cobra"
)

var (
	cmdDaemon = &cobra.Command{
		Use:   "daemon ",
		Short: "Run rly daemon.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initClient()
		},
	}
)
