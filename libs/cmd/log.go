package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"signmap/libs"
)

var logInfo = &cobra.Command{
	Use:   "log ",
	Short: "cat log",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		bt, err := os.ReadFile(libs.LogFile)
		if err == nil {
			os.Stdout.Write(bt)
		} else {
			println("Read log file error: ", err.Error())
		}
	},
}
