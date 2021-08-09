package sync_cmd

import (
	"fmt"
	"github.com/alexeyco/simpletable"
	"github.com/spf13/cobra"
	"signmap/libs"
	"strconv"
)

var (
	cmdInfo = &cobra.Command{
		Use:   "info",
		Short: "Get account info.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initClient()
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
		},
	}
)
