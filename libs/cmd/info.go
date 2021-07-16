package cmd

import (
	"github.com/spf13/cobra"
	"signmap/libs/contracts/matic_data"
)

var cmdInfo = &cobra.Command{
	Use:   "info ",
	Short: "Get user information",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		matic_data.GetData()
	},
}
