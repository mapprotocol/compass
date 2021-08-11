package cmd

import (
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/libs"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
)

var (
	cmdUnRegister = &cobra.Command{
		Use:   "unregister",
		Short: "With the unregister transaction executed, the unregistering portion is locked in the contract for about 2 epoch. After the period, you can withdraw the unregistered coins.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			var value big.Int
			var min = big.NewInt(100000)
			if len(args) == 0 {
				for {
					print("Enter the unregistering amount(Unit Wei):  ")
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
					log.Fatal("Value not a number ")
				}
				if value.Cmp(min) == -1 {
					log.Fatal("The value is at least 100000")
				}
			}
			cmd_runtime.InitClient()
			valueWei := libs.EthToWei(&value)
			if cmd_runtime.DstInstance.UnRegister(valueWei) {
				println("There are no errors, you can query by subcommand info to see if it was successful.")
			}
		},
	}
)
