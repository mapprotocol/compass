package cmd

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"signmap/libs"
	"signmap/libs/contracts/staking_bsc"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var (
	warnBalance = big.NewInt(1e16)
	doing       = false
	signUnit    int
	balance     *big.Int
	nowUnit     int
	date        string
	cmdDaemon   = &cobra.Command{
		Use:   "daemon ",
		Short: "(Default) Run signmap daemon .",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {

			// print the big label
			fmt.Println()
			fmt.Println(`    _____           __  ___           `)
			fmt.Println(`   / __(_)__ ____  /  |/  /__ ____    `)
			fmt.Println(`  _\ \/ / _ ` + "`" + `/ _ \/ /|_/ / _ ` + "`" + `/ _ \   `)
			fmt.Println(` /___/_/\_, /_//_/_/  /_/\_,_/ .__/   `)
			fmt.Println(`       /___/                /_/      BSC version`)
			fmt.Println()

			privateKey := libs.GetKey("")
			fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
			bindAddress := staking_bsc.BindAddress(fromAddress)
			if bytes.Equal(bindAddress.Bytes(), common.Address{}.Bytes()) {
				println("Worker not set！ please set a worker.")
				os.Exit(1)
			}
			amount := staking_bsc.GetData(bindAddress)
			if amount == nil || amount == big.NewInt(0) {
				println("No pledge yet, please pledge first.")
				os.Exit(1)
			}
			balance = libs.GetBalance(fromAddress)
			if balance.Cmp(warnBalance) == -1 {
				println("Lack of balance. The balance is： ", libs.WeiToEther(balance))
				os.Exit(1)
			}

			bindBalance := libs.GetBalance(bindAddress)
			// print info
			log.Println("Pledge Account: ", bindAddress.Hex())
			log.Println("Pledge Balance: ", libs.WeiToEther(bindBalance), "BNB")
			log.Println("Worker Account: ", fromAddress.Hex())
			log.Println("Worker Balance: ", libs.WeiToEther(balance), "BNB")
			rand.Seed(time.Now().UnixNano())
			libs.WriteLog(time.Now().Format("-20060102 15:04:05") + ". starting success!")
			log.Println("Start-up success")
			log.Println("Running process......")

			// rand a minute from now on to the end of the day
			signUnit, _ := libs.NowTime()
			diff := rand.Intn(24*60 - signUnit)
			signUnit += diff

			lastSignTimestamp, ok := staking_bsc.GetLastSign(bindAddress)
			if lastSignTimestamp.Int64() == 0 && ok {
				doing = true
				doSign(fromAddress, bindAddress)
			}
			c := make(chan bool)
			go func() {
				for {
					print("UTC time: ", time.Now().UTC().Format("15:04:05"), "\r")
					time.Sleep(time.Second)
				}
			}()
			go func(cc chan bool) {
				for {
					<-cc
					nowUnit, date = libs.NowTime() // for production
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
						lastSignTimestamp, _ = staking_bsc.GetLastSign(bindAddress)

						if libs.Unix2Time(*lastSignTimestamp).UTC().Format("20060102") != time.Now().UTC().Format("20060102") {
							doSign(fromAddress, bindAddress)
						} else {
							log.Println("Sign-in has been made today")
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

func doSign(workAddr common.Address, bindAddr common.Address) {
	log.Println("start signing.")
	go func() {
		if staking_bsc.DO() {
			libs.WriteLog(fmt.Sprintf("%s %d Sign in successfully.", date, nowUnit))
			signUnit = -1
			balance = libs.GetBalance(workAddr)
			if balance.Cmp(warnBalance) == -1 {
				log.Println("Lack of balance. The balance is： ", libs.WeiToEther(balance))
				log.Println("The next sign-in may fail, please recharge")
			}
			time.Sleep(10 * time.Second)
			staking_bsc.GetData(bindAddr)

		} else {
			log.Println("The server did not return the result,please check the status at the following website\nhttps://relayer.mapdapp.net/#/manage")
			libs.WriteLog(fmt.Sprintf("-%s %d unkown if it worked.", date, nowUnit))
		}
		doing = false
	}()
}
