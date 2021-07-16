package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"signmap/libs"
)

func Run() {
	setDefaultCommandIfNonePresent()
	var rootCmd = &cobra.Command{Use: "signmap"}
	rootCmd.AddCommand(cmdDaemon)
	rootCmd.AddCommand(cmdInfo)
	os.MkdirAll(libs.RuntimeDirectory, 0700)
	os.MkdirAll(libs.ConfigDirectory, 0700)

	rootCmd.Execute()
}
func setDefaultCommandIfNonePresent() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "daemon")
	}
}
