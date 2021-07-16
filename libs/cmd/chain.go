package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"signmap/libs"
)

var (
	cmdChain = &cobra.Command{
		Use:   "chain ",
		Short: "Configure the application chain. ",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	cmdChainLs = &cobra.Command{
		Use:   "ls ",
		Short: "List the application chain. ",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			printChainList()
		},
	}
	cmdChainAdd = &cobra.Command{
		Use:   "add ",
		Short: "Add a new  chain . ",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			newMap := libs.GetExternalBlockChainMap()
			println("Command-line input is not safe. Please guarantee that it is correct！")
			print("key: ")
			key := libs.ReadString()
			print("RpcUrl: ")
			RpcUrl := libs.ReadString()
			print("StakingContractAddress: ")
			StakingContractAddress := libs.ReadString()
			print("DataContractAddress: ")
			DataContractAddress := libs.ReadString()

			newMap[key] = libs.Chain{
				RpcUrl,
				StakingContractAddress,
				DataContractAddress,
			}
			bt, _ := json.Marshal(newMap)
			libs.WriteConfig(libs.ExternalBlockChainKey, string(bt))

		},
	}
	cmdChainDefault = &cobra.Command{
		Use:   "default ",
		Short: "set Default chain. ",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			var key string
			if len(args) != 0 {
				key = args[0]
			} else {
				print("key: ")
				key = libs.ReadString()
			}
			for {
				if _, ok := libs.GetBlockChainMap()[key]; ok {
					libs.WriteConfig("selected_chain", key)
					break
				} else {
					println("error option!")
				}
				printChainList()
				print("key: ")
				key = libs.ReadString()
			}
		},
	}
)

func cmdChainFunc() *cobra.Command {
	cmdChain.AddCommand(cmdChainLs, cmdChainAdd, cmdChainDefault)
	return cmdChain
}
func printChainList() {
	defaultChain := libs.ReadConfig("selected_chain", "1")

	//todo use go command table lib to beautify
	for k, v := range libs.GetBlockChainMap() {
		if k == defaultChain {
			fmt.Printf("(Default) %s = %+v", k, v)

		} else {
			fmt.Printf("%s = %+v", k, v)
		}
		println()
	}
}
