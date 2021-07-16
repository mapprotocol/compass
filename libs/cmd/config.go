package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"signmap/libs"
)

var configurable = map[string]bool{"keystore": true}

func cmdConfig() *cobra.Command {
	cmdConfig := &cobra.Command{
		Use:   "config ",
		Short: "Configure the application. ",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	configGet := &cobra.Command{
		Use:   "get ",
		Short: "Read the application configuration",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			for k := range configurable {
				println(k, "=", libs.ReadConfig(k, "Default"))
			}
		},
	}
	configSet := &cobra.Command{
		Use:   "set ",
		Short: "Write the application configuration",

		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("Error: requires at least 2 arg(s), only received 0")
				fmt.Println("Usage: ")
				fmt.Println("    signmap config set [key] [value]")
				return
			}
			if _, ok := configurable[args[0]]; ok {
				libs.WriteConfig(args[0], args[1])
				println(args[0], "=", args[1])
			} else {
				print("Only option (")
				for k := range configurable {
					print(k, ",")
				}
				println(") can be set.")
			}
		},
	}
	cmdConfig.AddCommand(configGet, configSet)
	return cmdConfig
}
