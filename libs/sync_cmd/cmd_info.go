package sync_cmd

import (
	"fmt"
	"github.com/alexeyco/simpletable"
	"github.com/spf13/cobra"
	"signmap/libs"
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
			initClient()
			displayOnce()
		},
	}
	cmdInfoWatch = &cobra.Command{
		Use:   "watch ",
		Short: "Get account info every some seconds ,default 5 seconds.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initClient()
			var interval = 5
			if len(args) != 0 {
				if i, err := strconv.Atoi(args[0]); err == nil {
					interval = i
				}
			}
			for {
				displayOnce()
				time.Sleep(time.Duration(interval) * time.Second)
			}
		},
	}
)

func cmdInfoFunc() *cobra.Command {
	cmdInfo.AddCommand(cmdInfoWatch)
	return cmdInfo
}
func displayOnce() {
	relayerBalance := dstInstance.GetRelayerBalance()
	relayer := dstInstance.GetRelayer()
	table := simpletable.New()
	table.Header = &simpletable.Header{
		Cells: []*simpletable.Cell{
			{Align: simpletable.AlignCenter, Text: "name"},
			{Align: simpletable.AlignCenter, Text: "value"},
		},
	}
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "registered amount"},
		{Text: libs.WeiToEther(relayerBalance.Register).String()},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "locked amount"},
		{Text: libs.WeiToEther(relayerBalance.Locked).String()},
	})
	table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
		{Text: "redeemable amount"},
		{Text: libs.WeiToEther(relayerBalance.Unlocked).String()},
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
	fmt.Println(table.String())
}
