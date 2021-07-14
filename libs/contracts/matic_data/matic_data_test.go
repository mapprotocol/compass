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
