package matic_data

import (
	"signmap/libs"
	"testing"
)

func TestGetData(t *testing.T) {
	libs.GetKey("123456")
	GetData()
}

func TestIsBindAddress(t *testing.T) {
	libs.GetKey("123456")
	println(BindAddress().Hex())
}

func TestGetLastSign(t *testing.T) {

	libs.GetKey("123456")
	signTimestamp := GetLastSign()
	tim := libs.Unix2Time(*signTimestamp)
	println(tim.Format("20060102"))
}
