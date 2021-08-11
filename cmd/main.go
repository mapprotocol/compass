package cmd

import (
	"github.com/spf13/cobra"
)

func Run() {
	var rootCmd = &cobra.Command{Use: "map_rly"}

	rootCmd.AddCommand(cmdRegister, cmdUnRegister, cmdInfoFunc(), cmdDaemon, cmdConfigFunc())
	err := rootCmd.Execute()
	if err != nil {
		return
	}
}
