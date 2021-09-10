package cmd

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/cmd/events"
	"github.com/mapprotocol/compass/http_call"
	"github.com/mapprotocol/compass/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/big"
	"strconv"
	"strings"
	"time"
)

var (
	srcBlockNumberByEstimation uint64 = 0
	dstBlockNumberByEstimation uint64 = 0
	getSrcBlockNumber                 = func() uint64 {
		if cmd_runtime.BlockNumberByEstimation {
			return srcBlockNumberByEstimation
		} else {
			return cmd_runtime.SrcInstance.GetBlockNumber()
		}
	}
	getDstBlockNumber = func() uint64 {
		if cmd_runtime.BlockNumberByEstimation {
			return dstBlockNumberByEstimation
		} else {
			return cmd_runtime.DstInstance.GetBlockNumber()
		}
	}
	currentWorkingBlockNumber uint64 = 0
	currentWorkedBlockNumber  uint64 = 0
	canDo                            = false
	cmdDaemon                        = &cobra.Command{
		Use:   "daemon ",
		Short: "Run rly daemon.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initDb()
			cmd_runtime.InitClient()
			updateCanDoThread()
			if cmd_runtime.BlockNumberByEstimation {
				updateBlockNumberThread(cmd_runtime.DstInstance, &dstBlockNumberByEstimation, 10)
				updateBlockNumberThread(cmd_runtime.SrcInstance, &srcBlockNumberByEstimation, 10)
			}
			listenEventThread()
			syncHeaderThread()
			for {
				time.Sleep(time.Hour)
			}
		},
	}
)

func syncHeaderThread() {
	go func() {
		for {
			if currentWorkedBlockNumber == 0 {
				time.Sleep(time.Second)
			} else {
				currentWorkingBlockNumber = currentWorkedBlockNumber
				break
			}
		}
		for {
			if !canDo {
				time.Sleep(time.Minute)
				continue
			}
			if currentWorkingBlockNumber+cmd_runtime.SrcInstance.GetStableBlockBeforeHeader() > getSrcBlockNumber() {
				cmd_runtime.DisplayMessageAndSleep(cmd_runtime.StructUnStableBlock)
				continue
			}
			byteData := cmd_runtime.SrcInstance.GetBlockHeader(currentWorkingBlockNumber)
			cmd_runtime.DstInstance.Save(cmd_runtime.SrcInstance.GetChainId(), byteData)
			currentWorkingBlockNumber += 1
		}
	}()
}

func listenEventThread() {
	log.Infoln("listenEventThread started.")
	eventSwapOutStr := utils.Get(levelDbInstance, cmd_runtime.EventSwapOutKey)
	eventSwapOutInt, _ := strconv.Atoi(eventSwapOutStr)
	eventSwapOutArrayStr := utils.Get(levelDbInstance, cmd_runtime.EventSwapOutArrayKey)
	var from = int64(eventSwapOutInt)
	if cmd_runtime.GlobalConfigV.StartWithBlock > from {
		from = cmd_runtime.GlobalConfigV.StartWithBlock
	}
	var to = from
	var i64SrcBlockNumber int64 = 0
	var lastBlockNumber = uint64(from)
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(from),
		ToBlock:   big.NewInt(to),
		Addresses: []common.Address{common.HexToAddress(cmd_runtime.SrcChainConfig.RouterContractAddress)},
	}
	go func() {
		for {
			i64SrcBlockNumber = int64(getSrcBlockNumber())

			if i64SrcBlockNumber-from <= int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader()) {
				time.Sleep(cmd_runtime.SrcInstance.NumberOfSecondsOfBlockCreationTime())
				continue
			}

			if i64SrcBlockNumber-from-int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader()) > 99 {
				to = from + 99
			} else {
				to = i64SrcBlockNumber - int64(cmd_runtime.SrcInstance.GetStableBlockBeforeHeader())
			}
			log.Infoln("queryEvent from:", from, ",to: ", to, ",block number:", i64SrcBlockNumber)

			query.FromBlock = big.NewInt(from)
			query.ToBlock = big.NewInt(to)

			logs, err := cmd_runtime.SrcInstance.GetClient().FilterLogs(context.Background(), query)
			log.Infoln("query ", len(logs), " events.")
			if err != nil {
				log.Warnln("cmd_runtime.SrcInstance.GetClient().FilterLogs error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			//var log types.Log
			for _, aLog := range logs {
				if cmd_runtime.EventSwapOutHash != aLog.Topics[0] {
					continue
				}
				if strings.Contains(eventSwapOutArrayStr, aLog.TxHash.String()) {
					continue
				}
				for {
					if aLog.BlockNumber > currentWorkedBlockNumber {
						_ = queryServerBlockNumber()
						time.Sleep(5 * time.Second)
					} else {
						break
					}
				}
				events.HandleLogSwapOut(&aLog)

				println()

				if aLog.BlockNumber != lastBlockNumber {
					utils.Put(levelDbInstance, cmd_runtime.EventSwapOutKey, strconv.Itoa(int(aLog.BlockNumber)))
					lastBlockNumber = aLog.BlockNumber
					eventSwapOutArrayStr = aLog.TxHash.String() + ","
				} else {
					eventSwapOutArrayStr += aLog.TxHash.String() + ","
				}
				utils.Put(levelDbInstance, cmd_runtime.EventSwapOutArrayKey, eventSwapOutArrayStr)

			}
			from = to + 1
			utils.Put(levelDbInstance, cmd_runtime.EventSwapOutKey, strconv.Itoa(int(from)))
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
			curDstBlockNumber := getDstBlockNumber()
			if getHeight.Relayer && getHeight.Start.Uint64() <= getDstBlockNumber() && getHeight.End.Uint64() >= curDstBlockNumber {
				if !canDo {
					//There is no room for errors when canDo convert from false to true
					if err := queryServerBlockNumber(); err != nil {
						log.Infoln("updateCurrentBlockNumber rpc call error")
						time.Sleep(time.Minute)
						continue
					}
				}
				canDo = true
				estimateTime := time.Duration((getHeight.End.Uint64()-curDstBlockNumber)/2) * cmd_runtime.DstInstance.NumberOfSecondsOfBlockCreationTime()
				if estimateTime > time.Minute {
					time.Sleep(estimateTime)
				} else {
					time.Sleep(time.Minute)
				}
			} else {
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
		var totalMilliseconds int64
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
func queryServerBlockNumber() error {

	headerCurrentNumber := http_call.HeaderCurrentNumber(cmd_runtime.DstInstance.GetRpcUrl(), cmd_runtime.SrcInstance.GetChainId())
	if headerCurrentNumber != ^uint64(0) && headerCurrentNumber >= currentWorkedBlockNumber {
		currentWorkedBlockNumber = headerCurrentNumber
		return nil
	}
	return fmt.Errorf("get currentWorkingBlockNumber err")
}
