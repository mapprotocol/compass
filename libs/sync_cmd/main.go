package sync_cmd

import (
	"github.com/spf13/cobra"
	"signmap/libs/sync_libs/chain_structs"
)

func Run() {
	var rootCmd = &cobra.Command{Use: "map_rly"}

	err := rootCmd.Execute()
	if err != nil {
		return
	}
	v := chain_structs.ChainId2Instance[1]
	println(v.GetBlockNumber())
}
