package cmd

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/http_call"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
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

			cmd_runtime.InitClient()

			updateCanDoThread()
			updateBlockNumberThread(cmd_runtime.DstInstance, &dstBlockNumber, 1)
			updateBlockNumberThread(cmd_runtime.SrcInstance, &srcBlockNumber, 1)
			updateCurrentBlockNumberThread()
			//listenEvenThread()
			for {
				//println(srcBlockNumber,dstBlockNumber)  // Reserve for testing
				if !canDo {
					time.Sleep(time.Minute)
					continue
				}
				if currentBlockNumber+cmd_runtime.SrcInstance.GetStableBlockBeforeHeader() > srcBlockNumber {
					cmd_runtime.DisplayMessageAndSleep(cmd_runtime.StructUnStableBlock)
					continue
				}
				byteData := cmd_runtime.SrcInstance.GetBlockHeader(currentBlockNumber)
				cmd_runtime.DstInstance.Save(cmd_runtime.SrcInstance.GetChainId(), byteData)
				currentBlockNumber += 1
			}
		},
	}
)

func listenEvenThread() {
	var from int64 = 0
	var to int64 = 0
	var i64SrcBlockNumber int64 = 0
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(to),
		Addresses: []common.Address{common.HexToAddress("0xb61119f02Eb017282b799f1120c57B415F2FAD6c")},
	}
	go func() {
		for {
			i64SrcBlockNumber = int64(cmd_runtime.SrcInstance.GetBlockNumber())

			if i64SrcBlockNumber-from <= int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader()) {
				time.Sleep(cmd_runtime.SrcInstance.NumberOfSecondsOfBlockCreationTime())
				continue
			}
			println(i64SrcBlockNumber, from, cmd_runtime.SrcInstance.GetStableBlockBeforeHeader())

			if i64SrcBlockNumber-from-int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader()) > 100 {
				to = from + 100
			} else {
				to = i64SrcBlockNumber - int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader())
			}

			query.ToBlock = big.NewInt(to)

			_, err := cmd_runtime.SrcInstance.GetClient().FilterLogs(context.Background(), query)
			if err != nil {
				log.Warnln("cmd_runtime.SrcInstance.GetClient().FilterLogs error", err)
				continue
			}
			from = to
			query.FromBlock = big.NewInt(from)
		}

	}()
}

func updateCanDoThread() {
	go func() {
		for {
			relayer := cmd_runtime.DstInstance.GetRelayer()
			if !relayer.Register {
				canDo = false
				cmd_runtime.DisplayMessageAndSleep(cmd_runtime.StructUnregistered)
				continue
			}
			if !relayer.Relayer {
				canDo = false
				cmd_runtime.DisplayMessageAndSleep(cmd_runtime.StructRegisterNotRelayer)
				continue
			}
			getHeight := cmd_runtime.DstInstance.GetPeriodHeight()

			if getHeight.Relayer && getHeight.Start.Uint64() <= dstBlockNumber && getHeight.End.Uint64() >= dstBlockNumber {
				if !canDo {
					//There is no room for errors when canDo convert from false to true
					if updateCurrentBlockNumber() == ^uint64(0) {
						log.Infoln("updateCurrentBlockNumber rpc call error")
						time.Sleep(time.Minute)
						continue
					}
				}
				canDo = true
				estimateTime := time.Duration((getHeight.End.Uint64()-dstBlockNumber)/2) * cmd_runtime.DstInstance.NumberOfSecondsOfBlockCreationTime()
				if estimateTime > time.Minute {
					time.Sleep(estimateTime)
				} else {
					time.Sleep(time.Minute)
				}
			} else {
				println("start end :", getHeight.Start.Uint64(), getHeight.End.Uint64())
				println("dst block number", dstBlockNumber)
				canDo = false
				time.Sleep(time.Minute)
			}
		}
	}()
}
func updateBlockNumberThread(chainImpl chains.ChainInterface, blockNumber *uint64, times int) {
	go func() {
		var i = 1
		var interval = chainImpl.NumberOfSecondsOfBlockCreationTime()
		var totalMilliseconds int64 = 0
		var startBlockNumber = chainImpl.GetBlockNumber()
		*blockNumber = startBlockNumber
		var startTime = time.Now().UnixNano()
		for {
			if canDo && i%times == 0 {
				byIncr := *blockNumber
				*blockNumber = chainImpl.GetBlockNumber()
				totalMilliseconds = time.Now().UnixNano() - startTime
				if *blockNumber == startBlockNumber {
					if interval*2 < chainImpl.NumberOfSecondsOfBlockCreationTime() {
						log.Infoln("interval is too small，It should be close to",
							chainImpl.NumberOfSecondsOfBlockCreationTime().String(),
							". It's actually ", interval.String())
					} else if interval > chainImpl.NumberOfSecondsOfBlockCreationTime()*2 {
						log.Infoln("interval is too big，It should be close to",
							chainImpl.NumberOfSecondsOfBlockCreationTime().String(),
							". It's actually ", interval.String())
					}
					log.Infoln("block number not change")
					i += 1
					time.Sleep(interval)
					continue
				}
				interval = time.Duration(uint64(totalMilliseconds) / (*blockNumber - startBlockNumber))
				log.Infoln(chainImpl.GetName(), "block number : byIncr =", byIncr, ", byRpc =", *blockNumber)
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
	headerCurrentNumber := http_call.HeaderCurrentNumber(cmd_runtime.DstInstance.GetRpcUrl(), cmd_runtime.SrcInstance.GetChainId())
	if headerCurrentNumber != ^uint64(0) && headerCurrentNumber > currentBlockNumber {
		currentBlockNumber = headerCurrentNumber + 1
	}
	log.Infoln("headerCurrentNumber =", headerCurrentNumber)
	return headerCurrentNumber
}
