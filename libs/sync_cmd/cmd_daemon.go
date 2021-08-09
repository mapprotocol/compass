package sync_cmd

import (
	"github.com/spf13/cobra"
	"time"
)

type waitTimeAndMessage struct {
	Time    time.Duration
	Message string
}

var (
	structRegisterNotRelayer = &waitTimeAndMessage{
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
			for {
				if !canDo {
					time.Sleep(time.Minute)
				}
				println("test test test")
				time.Sleep(2 * time.Second)
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
		}
	}()
}
