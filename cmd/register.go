package cmd

import (
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/libs"
	"github.com/spf13/cobra"
	"math/big"
	"os"
)

var (
	cmdRegister = &cobra.Command{
		Use:   "register",
		Short: "To become relayer, you need to register with some eth coins.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			var value big.Int
			var min = big.NewInt(100000)
			if len(args) == 0 {
				for {
					print("Enter the registering amount(Unit Eth):  ")
					valueString := libs.ReadString()
					if _, ok := value.SetString(valueString, 10); ok {
						if value.Cmp(min) == -1 {
							println("The value is at least 100000")
							continue
						}
						break
					} else {
						println("Not a number ")
					}
				}
			} else {
				if _, ok := value.SetString(args[0], 10); !ok {
					println("Not a number ")
					os.Exit(1)
				}
				if value.Cmp(min) == -1 {
					println("The value is at least 100000")
					os.Exit(1)
				}
			}

			cmd_runtime.InitClient()
			valueWei := libs.EthToWei(&value)
			if cmd_runtime.DstInstance.Register(valueWei) {
				println("There are no errors, you can query by subcommand info to see if it was successful.")
			}
		},
	}
)
