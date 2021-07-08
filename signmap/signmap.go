package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"signmap/libs"
	"signmap/libs/contracts/MaticStaking"
	"strings"
	"time"
)

func main() {
	p := flag.String("password", "", "Enter the password on the command line ")
	flag.Parse()

	libs.GetKey(*p)

	rand.Seed(time.Now().UnixNano())
	//signUnit := rand.Intn(24 * 60) //for production
	libs.WriteLog("starting success!")
	var everyNMinute = 60               // require 60 % everyNMinute == 0 //for test
	signUnit := rand.Intn(everyNMinute) //for test
	log.Println("signUnit = ", signUnit)

	for {
		go func() {
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
				MaticStaking.DO()
			}
		}()

		time.Sleep(time.Minute)
	}
}
