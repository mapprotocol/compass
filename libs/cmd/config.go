package cmd

import (
	"fmt"
	"github.com/mapprotocol/compass/libs"
	"github.com/spf13/cobra"
)

var (
	configurable = map[string]bool{"keystore": true}
	cmdConfig    = &cobra.Command{
		Use:   "config ",
		Short: "Configure the application.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				return
			}
		},
	}
	configGet = &cobra.Command{
		Use:   "get ",
		Short: "Read the application configuration.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			for k := range configurable {
				println(k, "=", libs.ReadConfig(k, "Default"))
			}
		},
	}
	configSet = &cobra.Command{
		Use:   "set ",
		Short: "Write the application configuration.",

		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Println("Error: requires at least 2 args, only received ", len(args), ".")
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
	configErase = &cobra.Command{
		Use:   "erase ",
		Short: "Erase the application configuration key ",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if _, ok := configurable[args[0]]; ok {
				libs.EraseConfig(args[0])
			} else {
				print("Only option (")
				for k := range configurable {
					print(k, ",")
				}
				println(") can be erased.")
			}
		},
	}
	configReset = &cobra.Command{
		Use:   "reset ",
		Short: "Reset the application configuration to default ",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			for k := range configurable {
				libs.EraseConfig(k)

			}
		},
	}
)

func cmdConfigFunc() *cobra.Command {

	cmdConfig.AddCommand(configGet, configSet, configErase, configReset)
	return cmdConfig
}
