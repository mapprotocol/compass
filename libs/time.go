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
