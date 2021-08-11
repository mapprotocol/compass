package cmd

import (
	"github.com/mapprotocol/compass/cmd/common"
	"github.com/mapprotocol/compass/libs/sync_libs"
	"github.com/mapprotocol/compass/libs/sync_libs/chain_structs"
	"github.com/spf13/cobra"
	"log"
	"time"
)

var (
	srcBlockNumber     uint64 = 0
	dstBlockNumber     uint64 = 0
	currentBlockNumber uint64 = 0
	canDo                     = false
	cmdDaemon                 = &cobra.Command{
		Use:   "daemon ",
		Short: "Run rly daemon.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			common.InitClient()
			updateCanDoThread()
			updateBlockNumberThread(common.DstInstance, &dstBlockNumber, 20)
			updateBlockNumberThread(common.SrcInstance, &srcBlockNumber, 10)
			updateCurrentBlockNumberThread()

			for {
				//println(srcBlockNumber,dstBlockNumber)  // Reserve for testing
				if !canDo {
					time.Sleep(time.Minute)
					continue
				}
				if currentBlockNumber+common.SrcInstance.GetStableBlockBeforeHeader() > srcBlockNumber {
					common.DisplayMessageAndSleep(common.StructUnStableBlock)
					continue
				}
				byteData := common.SrcInstance.GetBlockHeader(currentBlockNumber)
				common.DstInstance.SyncBlock(common.SrcInstance.GetChainEnum(), byteData)
				currentBlockNumber += 1
			}
		},
	}
)

func updateCanDoThread() {
	go func() {
		for {
			relayer := common.DstInstance.GetRelayer()
			if !relayer.Register {
				canDo = false
				common.DisplayMessageAndSleep(common.StructUnregistered)
				continue
			}
			if !relayer.Relayer {
				canDo = false
				common.DisplayMessageAndSleep(common.StructRegisterNotRelayer)
				continue
			}
			getHeight := common.DstInstance.GetPeriodHeight()
			//println("start end :",getHeight.Start.Uint64(),getHeight.End.Uint64())
			//println("dst block number", dstBlockNumber)
			if getHeight.Relayer && getHeight.Start.Uint64() <= dstBlockNumber && getHeight.End.Uint64() >= dstBlockNumber {
				if !canDo {
					//There is no room for errors when canDo convert from false to true
					if updateCurrentBlockNumber() == ^uint64(0) {
						log.Println("updateCurrentBlockNumber rpc call error")
						time.Sleep(time.Minute)
						continue
					}
				}
				canDo = true
				estimateTime := time.Duration((getHeight.End.Uint64()-dstBlockNumber)/2) * common.DstInstance.NumberOfSecondsOfBlockCreationTime()
				if estimateTime > 5*time.Minute {
					time.Sleep(estimateTime)
				} else {
					time.Sleep(5 * time.Minute)
				}
			} else {
				canDo = false
				time.Sleep(5 * time.Minute)
			}
		}
	}()
}
func updateBlockNumberThread(chainImpl chain_structs.ChainInterface, blockNumber *uint64, times int) {
	go func() {
		var i = 1
		var interval = chainImpl.NumberOfSecondsOfBlockCreationTime()
		var totalMilliseconds int64 = 0
		var startBlockNumber = chainImpl.GetBlockNumber()
		*blockNumber = startBlockNumber
		var startTime = time.Now().UnixNano()
		for {
			if canDo && i%times == 0 {
				*blockNumber = chainImpl.GetBlockNumber()
				totalMilliseconds = time.Now().UnixNano() - startTime
				if *blockNumber == startBlockNumber {
					if interval*2 < chainImpl.NumberOfSecondsOfBlockCreationTime() {
						log.Println("interval is too small，It should be close to",
							chainImpl.NumberOfSecondsOfBlockCreationTime().String(),
							". It's actually ", interval.String())
					} else if interval > chainImpl.NumberOfSecondsOfBlockCreationTime()*2 {
						log.Println("interval is too big，It should be close to",
							chainImpl.NumberOfSecondsOfBlockCreationTime().String(),
							". It's actually ", interval.String())
					}
					log.Println("block number not change")
					i += 1
					time.Sleep(interval)
					continue
				}
				interval = time.Duration(uint64(totalMilliseconds) / (*blockNumber - startBlockNumber))
				log.Println(chainImpl.GetName(), ":", *blockNumber)
			} else {
				// if !canDo ,this number is very different from the true value, but it doesn't matter.
				*blockNumber += 1
			}
			i += 1
			time.Sleep(interval)
		}
	}()
}

func updateCurrentBlockNumberThread() {
	updateCurrentBlockNumber()
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if canDo {
				updateCurrentBlockNumber()
			}
		}
	}()
}

func updateCurrentBlockNumber() uint64 {
	headerCurrentNumber := sync_libs.HeaderCurrentNumber(common.SrcInstance.GetRpcUrl(), common.SrcInstance.GetChainEnum())
	if headerCurrentNumber != ^uint64(0) && headerCurrentNumber > currentBlockNumber {
		currentBlockNumber = headerCurrentNumber + 1
	}
	return headerCurrentNumber
}
