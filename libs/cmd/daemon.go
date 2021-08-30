package cmd

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"log"
	"math/big"
	"math/rand"
	"os"
	"signmap/libs"
	"signmap/libs/contracts/matic_data"
	"signmap/libs/contracts/matic_staking"
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
		amount := matic_data.GetData()
		if amount == nil || amount == big.NewInt(0) {
			println("No pledge yet, please pledge first.")
			os.Exit(1)
		}
		rand.Seed(time.Now().UnixNano())
		libs.WriteLog(time.Now().Format("20060102 15:04:05") + ". starting success!")
		log.Println("Start-up success")
		log.Println("Running process......")

		signUnit := rand.Intn(24 * 60) //for production
		//var everyNMinute = 1                 // require 60 % everyNMinute == 0 //for test
		//signUnit := rand.Intn(everyNMinute)  //for test
		//log.Println("signUnit = ", signUnit) // for test , production environment does not print
		c := make(chan bool)
		go func() {
			for {
				print("\r", "UTC time: ", time.Now().UTC().Format("15:04:05"))
				time.Sleep(time.Second)
			}
		}()
		go func(cc chan bool) {
			for {
				_ = <-cc
				nowUnit, date := libs.NowTime() // for production
				//nowUnit, date := libs.NowTimeForTestEveryNMinute(everyNMinute) //for test
				if nowUnit == 0 {
					//Since the sleep is less than 1 minute, this may be performed twice one day
					//If signUnit is not specified as 0, there will be no problem
					signUnit = rand.Intn(24*60-1) + 1 //for production
					//signUnit = rand.Intn(everyNMinute) //for test
					log.Println("signUnit = ", signUnit)
				}

				if nowUnit == signUnit {
					log.Println("start signing. step 1.")
					// Determine if you have signed it today
					if !strings.HasPrefix(libs.GetLastLineWithSeek(), date) {
						log.Println("start signing. step 2.")
						go func() {
							if matic_staking.DO() {
								libs.WriteLog(fmt.Sprintf("%s %d Sign in successfully.", date, nowUnit))
								matic_data.GetData()
								signUnit = -1
							} else {
								// add - let strings.HasPrefix(libs.GetLastLineWithSeek() return false
								libs.WriteLog(fmt.Sprintf("-%s %d Sign in unsuccessfully.", date, nowUnit))
							}
						}()
					}
				}
			}
		}(c)
		for {
			c <- true
			time.Sleep(time.Second * 55)
		}
	},
}
