package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"signmap/libs"
	"signmap/libs/contracts/matic_data"
	"signmap/libs/contracts/matic_staking"
	"strings"
	"time"
)

func main() {
	p := flag.String("password", "", "Enter the password on the command line. not recommend. ")
	flag.Parse()
	libs.GetKey(*p)

	rand.Seed(time.Now().UnixNano())
	libs.WriteLog("starting success!")
	//signUnit := rand.Intn(24 * 60) //for production
	var everyNMinute = 1                 // require 60 % everyNMinute == 0 //for test
	signUnit := rand.Intn(everyNMinute)  //for test
	log.Println("signUnit = ", signUnit) // for test , production environment does not print
	c := make(chan bool)
	go func(cc chan bool) {
		for {
			_ = <-cc
			//nowTime,date := libs.NowTime()         // for production
			nowUnit, date := libs.NowTimeForTestEveryNMinute(everyNMinute) //for test
			if nowUnit == 0 {
				//signUnit = rand.Intn(24 * 60) //for production
				signUnit = rand.Intn(everyNMinute) //for test
				log.Println("signUnit = ", signUnit)
			}

			if nowUnit == signUnit && !strings.HasPrefix(libs.GetLastLineWithSeek(), date) {
				log.Println(date)
				// Determine if you have signed it today
				libs.WriteLog(fmt.Sprintf("%s %d Sign in successfully.", date, nowUnit))
				//libs.SendTransaction()
				if matic_staking.DO() {
					matic_data.GetData()
				}
			}
		}
	}(c)
	for {
		c <- true
		time.Sleep(time.Minute)
	}
}
