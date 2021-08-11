package cmd

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts/matic_data"
	"github.com/mapprotocol/compass/libs/contracts/matic_staking"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"strings"
	"time"
)

var cmdDaemon = &cobra.Command{
	Use:   "daemon ",
	Short: "(Default) Run signmap daemon .",
	Args:  cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		libs.GetKey("")
		if bytes.Compare(matic_data.BindAddress().Bytes(), common.Address{}.Bytes()) == 0 {
			println("Worker not set！ please set a worker.")
			os.Exit(1)
		}
		matic_data.GetData()
		rand.Seed(time.Now().UnixNano())
		libs.WriteLog(time.Now().Format("20060102 15:04:05") + ". starting success!")
		signUnit := rand.Intn(24 * 60) //for production
		//var everyNMinute = 1                 // require 60 % everyNMinute == 0 //for test
		//signUnit := rand.Intn(everyNMinute)  //for test
		//log.Infoln("signUnit = ", signUnit) // for test , production environment does not print
		c := make(chan bool)
		go func(cc chan bool) {
			for {
				_ = <-cc
				nowUnit, date := libs.NowTime() // for production
				//nowUnit, date := libs.NowTimeForTestEveryNMinute(everyNMinute) //for test
				if nowUnit == 0 {
					signUnit = rand.Intn(24 * 60) //for production
					//signUnit = rand.Intn(everyNMinute) //for test
					log.Infoln("signUnit = ", signUnit)
				}

				if nowUnit == signUnit && !strings.HasPrefix(libs.GetLastLineWithSeek(), date) {
					log.Infoln(date)
					// Determine if you have signed it today
					if matic_staking.DO() {
						libs.WriteLog(fmt.Sprintf("%s %d Sign in successfully.", date, nowUnit))
						matic_data.GetData()
					}
				}
			}
		}(c)
		for {
			c <- true
			time.Sleep(time.Minute)
		}
	},
}
