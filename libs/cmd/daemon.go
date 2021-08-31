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
	"time"
)

var (
	warnBalance = big.NewInt(1e16)
	doing       = false
	cmdDaemon   = &cobra.Command{
		Use:   "daemon ",
		Short: "(Default) Run signmap daemon .",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			libs.GetKey("")
			if bytes.Equal(matic_data.BindAddress().Bytes(), common.Address{}.Bytes()) {
				println("Worker not set！ please set a worker.")
				os.Exit(1)
			}
			amount := matic_data.GetData()
			if amount == nil || amount == big.NewInt(0) {
				println("No pledge yet, please pledge first.")
				os.Exit(1)
			}
			balance := libs.GetBalance()
			if balance.Cmp(warnBalance) == -1 {
				println("Lack of balance. The balance is： ", libs.WeiToEther(balance))
				os.Exit(1)
			}
			rand.Seed(time.Now().UnixNano())
			libs.WriteLog(time.Now().Format("-20060102 15:04:05") + ". starting success!")
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
					<-cc
					nowUnit, date := libs.NowTime() // for production
					if nowUnit == 0 {
						//Since the sleep is less than 1 minute, this may be performed twice one day
						//If signUnit is not specified as 0, there will be no problem
						signUnit = rand.Intn(24*60-1) + 1 //for production
						log.Println("signUnit = ", signUnit)
						doing = false //Fault tolerance
					}

					if nowUnit == signUnit && !doing {
						doing = true

						// Determine if you have signed it today
						if libs.Unix2Time(*matic_data.GetLastSign()).Format("20060102") != time.Now().UTC().Format("20060102") {
							log.Println("start signing.")
							go func() {
								if matic_staking.DO() {
									libs.WriteLog(fmt.Sprintf("%s %d Sign in successfully.", date, nowUnit))
									matic_data.GetData()
									signUnit = -1
									balance = libs.GetBalance()
									if balance.Cmp(warnBalance) == -1 {
										log.Println("Lack of balance. The balance is： ", libs.WeiToEther(balance))
										log.Println("The next sign-in may fail, please recharge")
									}
								} else {
									// add - let strings.HasPrefix(libs.GetLastLineWithSeek() return false
									log.Println("Sign in unsuccessfully.")
									libs.WriteLog(fmt.Sprintf("-%s %d Sign in unsuccessfully.", date, nowUnit))
								}
								doing = false
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
)
