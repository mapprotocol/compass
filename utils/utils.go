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
