package sync_cmd

import (
	"github.com/spf13/cobra"
	"log"
	"signmap/libs/sync_libs/chain_structs"
	"time"
)

type waitTimeAndMessage struct {
	Time    time.Duration
	Message string
}

var (
	srcBlockNumber           uint64 = 0
	dstBlockNumber           uint64 = 0
	structRegisterNotRelayer        = &waitTimeAndMessage{
		Time:    time.Minute,
		Message: "",
	}
	structUnregistered = &waitTimeAndMessage{
		Time:    10 * time.Minute,
		Message: "structUnregistered",
	}

	cmdDaemon = &cobra.Command{
		Use:   "daemon ",
		Short: "Run rly daemon.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			initClient()
			updateCanDoThread()
			updateBlockNumber(dstInstance, &dstBlockNumber, 10)
			updateBlockNumber(srcInstance, &srcBlockNumber, 5)
			for {
				//println(srcBlockNumber,dstBlockNumber)  // Reserve for testing
				time.Sleep(time.Second * 5)

			}
		},
	}
	canDo = false
)

func displayMessageAndSleep(s *waitTimeAndMessage) {
	println(s.Message)
	time.Sleep(s.Time)
}
func updateCanDoThread() {
	go func() {
		for {
			relayer := dstInstance.GetRelayer()
			if !relayer.Register {
				canDo = false
				displayMessageAndSleep(structUnregistered)
				continue
			}
			if !relayer.Relayer {
				canDo = false
				displayMessageAndSleep(structRegisterNotRelayer)
				continue
			}
			getHeight := dstInstance.GetPeriodHeight()
			if getHeight.Relayer {
				canDo = true

			} else {
				// should unreachable
				log.Println("blockchain error !")
				canDo = false
				time.Sleep(5 * time.Minute)
			}
		}
	}()
}
func updateBlockNumber(chainImpl chain_structs.ChainInterface, blockNumber *uint64, times int) {
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
					println("block number not change")
					i += 1
					time.Sleep(interval)
					continue
				}
				interval = time.Duration(uint64(totalMilliseconds) / (*blockNumber - startBlockNumber))
				log.Println(chainImpl.GetName(), ":", *blockNumber)
			} else {
				*blockNumber += 1
			}
			i += 1
			time.Sleep(interval)
		}
	}()
}
