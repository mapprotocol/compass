package cmd

import (
	"github.com/mapprotocol/compass/libs/contracts/matic_data"
	"github.com/spf13/cobra"
)

var cmdInfo = &cobra.Command{
	Use:   "info ",
	Short: "Get user information",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		matic_data.GetData()
	},
}
