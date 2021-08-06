package sync_cmd

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var (
	cmdInfo = &cobra.Command{
		Use:   "info",
		Short: "Get account info.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initClient()
			r := dstInstance.GetRelayerBalance()
			fmt.Printf("%+v", r)
			println(common.BytesToAddress([]byte("RelayerAddress")).String())

		},
	}
)
