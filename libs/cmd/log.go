package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"signmap/libs"
)

var cmdLog = &cobra.Command{
	Use:   "log ",
	Short: "Show sign log",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		bt, err := os.ReadFile(libs.LogFile)
		if err == nil {
			_, err := os.Stdout.Write(bt)
			if err != nil {
				return
			}
		} else {
			println("Read log file error: ", err.Error())
		}
	},
}
