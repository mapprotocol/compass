package utils

import (
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	log "github.com/sirupsen/logrus"
	"math/big"
)

func ReadString() string {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		log.Fatal(err)
	}
	return input
}

func WeiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}
func EthToWei(Value *big.Int) *big.Int {
	baseUnit := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	value := new(big.Int).Mul(Value, baseUnit)
	return value
}
func ByteArray2Byte32Array(input *[]byte) *[][32]byte {
	var output = make([][32]byte, 0)
	var cur = 0
	var byteLen = len(*input)
	for {
		if cur+32 >= byteLen {
			break
		}
		curByte := [32]byte{}
		copy(curByte[:], (*input)[cur:cur+32])
		output = append(output, curByte)
		cur += 32
	}
	curByte := [32]byte{}
	copy(curByte[:], (*input)[cur:])
	output = append(output, curByte)
	return &output
}
