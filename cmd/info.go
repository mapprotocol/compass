package cmd

import (
	"fmt"
	"github.com/alexeyco/simpletable"
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"strconv"
	"time"
)

var (
	cmdInfo = &cobra.Command{
		Use:   "info",
		Short: "Get account info.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				err := cmd.Help()
				if err != nil {
					return
				}
				return
			}
			cmd_runtime.InitClient()
			displayOnce(false)
		},
	}
	cmdInfoWatch = &cobra.Command{
		Use:   "watch ",
		Short: "Get account info every some seconds ,default 5 seconds.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			cmd_runtime.InitClient()
			var interval = 5
			if len(args) != 0 {
				if i, err := strconv.Atoi(args[0]); err == nil {
					interval = i
				}
			}
			for {
				displayOnce(true)
				time.Sleep(time.Duration(interval) * time.Second)
			}
		},
	}
)

func cmdInfoFunc() *cobra.Command {
	cmdInfo.AddCommand(cmdInfoWatch)
	return cmdInfo
}
func displayOnce(clearScreen bool) {
	relayerBalance := cmd_runtime.DstInstance.GetRelayerBalance()
	relayer := cmd_runtime.DstInstance.GetRelayer()
	if relayerBalance.Registered == nil || relayer.Epoch == nil {
		log.Warnln("call GetRelayerBalance or GetRelayer return nil")
		return
	}

	table := simpletable.New()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "name"},
			{Align: simpletable.AlignCenter, Text: "value"},
		},
	}
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "registered amount"},
		{Text: utils.WeiToEther(relayerBalance.Registered).String()},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "locked amount"},
		{Text: utils.WeiToEther(relayerBalance.Unregistering).String()},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "redeemable amount"},
		{Text: utils.WeiToEther(relayerBalance.Unregistered).String()},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "is registered"},
		{Text: strconv.FormatBool(relayer.Register)},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "is relayer"},
		{Text: strconv.FormatBool(relayer.Relayer)},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "current epoch"},
		{Text: relayer.Epoch.String()},
	})
	if clearScreen {
		cmd_runtime.CallClear()
	}
	fmt.Println(table.String())
}
