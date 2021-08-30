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
		Short: "Configure the application chain.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	cmdChainLs = &cobra.Command{
		Use:   "ls ",
		Short: "Show the chain list.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			printChainList()
		},
	}
	cmdChainAdd = &cobra.Command{
		Use:   "add ",
		Short: "Add/Update a chain.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			newMap := libs.GetExternalBlockChainMap()
			println("User input is not safe. Please guarantee that it is correct by yourself！")
			print("key: ")
			key := libs.ReadString()
			print("RpcUrl: ")
			RpcUrl := libs.ReadString()
			print("StakingContractAddress: ")
			StakingContractAddress := libs.ReadString()
			print("DataContractAddress: ")
			DataContractAddress := libs.ReadString()

			newMap[key] = libs.Chain{
				RpcUrl:                 RpcUrl,
				StakingContractAddress: StakingContractAddress,
				DataContractAddress:    DataContractAddress,
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
				printChainList()
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
	cmdChainDel = &cobra.Command{
		Use:   "del ",
		Short: "Del Use Input chain. ",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			newMap := libs.GetExternalBlockChainMap()
			if len(newMap) == 0 {
				println("Nothing to delete.")
				return
			}
			var key string
			if len(args) != 0 {
				key = args[0]
			} else {
				printUserInputChainList()
				print("key: ")
				key = libs.ReadString()
			}
			for {
				if _, ok := libs.GetBlockChainMap()[key]; ok {

					delete(newMap, key)
					bt, _ := json.Marshal(newMap)
					libs.WriteConfig(libs.ExternalBlockChainKey, string(bt))
					break
				} else {
					println("error option!")
				}
				printUserInputChainList()
				print("key: ")
				key = libs.ReadString()
			}
		},
	}
)

func cmdChainFunc() *cobra.Command {
	cmdChain.AddCommand(cmdChainLs, cmdChainAdd, cmdChainDefault, cmdChainDel)
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
func printUserInputChainList() {
	//todo use go command table lib to beautify
	for k, v := range libs.GetExternalBlockChainMap() {
		fmt.Printf("%s = %+v", k, v)
		println()
	}
}
