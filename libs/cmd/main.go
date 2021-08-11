package cmd

import (
	"github.com/mapprotocol/compass/libs"
	"github.com/spf13/cobra"
	"os"
)

func Run() {
	setDefaultCommandIfNonePresent()
	var rootCmd = &cobra.Command{Use: "signmap"}
	rootCmd.AddCommand(cmdDaemon)
	rootCmd.AddCommand(cmdInfo)
	rootCmd.AddCommand(cmdLog)
	rootCmd.AddCommand(cmdConfigFunc())
	rootCmd.AddCommand(cmdChainFunc())
	err := os.MkdirAll(libs.RuntimeDirectory, 0700)
	if err != nil {
		return
	}
	err = os.MkdirAll(libs.ConfigDirectory, 0700)
	if err != nil {
		return
	}

	err = rootCmd.Execute()
	if err != nil {
		return
	}
}
func setDefaultCommandIfNonePresent() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "daemon")
	}
}
