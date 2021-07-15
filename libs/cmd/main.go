package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

func Run() {
	setDefaultCommandIfNonePresent()
	var rootCmd = &cobra.Command{Use: "signmap"}
	rootCmd.AddCommand(cmdDaemon)
	rootCmd.Execute()
}
func setDefaultCommandIfNonePresent() {
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "daemon")
	}
}
