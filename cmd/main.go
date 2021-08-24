package cmd

import (
	"github.com/spf13/cobra"
)

func Run() {
	var rootCmd = &cobra.Command{Use: "map-rly"}

	rootCmd.AddCommand(cmdRegister, cmdUnRegister, cmdInfoFunc(), cmdDaemon, passwordCmd)
	err := rootCmd.Execute()
	if err != nil {
		return
	}
}
