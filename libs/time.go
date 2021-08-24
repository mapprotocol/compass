package libs

import (
	"fmt"
	"time"
)

func NowTime() (int, string) {
	now := time.Now().UTC()

	return now.Hour()*60 + now.Minute(), now.Format("20060102")
}

func NowTimeForTestEveryNMinute(n int) (int, string) {
	now := time.Now()
	return now.Minute() % n, now.Format("2006010215") + fmt.Sprint(now.Minute()/n)
}
