package libs

import (
	"time"
)

func NowTime() (int, string) {
	now := time.Now()
	return now.Hour()*60 + now.Minute(), now.Format("20060102")
}
func NowTimeForTest() (int, string) {
	now := time.Now()
	return now.Minute(), now.Format("2006010215")
}
func NowTimeForTestEveryNMinute(n int) (int, string) {
	now := time.Now()
	return now.Minute() % n, now.Format("2006010215") + string(now.Minute()/n)
}
