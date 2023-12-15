package util

import (
	"encoding/hex"
	"math/bits"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func Key2Hex(str []byte, proofLength int) []byte {
	ret := make([]byte, 0)
	for _, b := range str {
		ret = append(ret, b/16)
		ret = append(ret, b%16)
	}
	return ret
}

// FromHexString returns a byte array given a hex string
func FromHexString(data string) []byte {
	data = strings.TrimPrefix(data, "0x")
	if len(data)%2 == 1 {
		// Odd number of characters; even it up
		data = "0" + data
	}
	ret, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return ret
}

type Bitvector512 []byte

const bitvector512ByteSize = 64
const bitvector512BitSize = bitvector512ByteSize * 8

func NewBitvector512(data []byte) Bitvector512 {
	if len(data) != bitvector512ByteSize {
		return nil
	}
	byteArray := make([]byte, 0, bitvector512ByteSize)
	byteArray = append(byteArray, data...)
	return byteArray[:]
}

// Len returns the number of bits in the bitvector.
func (b Bitvector512) Len() uint64 {
	return bitvector512BitSize
}

// Count returns the number of 1s in the bitvector.
func (b Bitvector512) Count() uint64 {
	if len(b) == 0 {
		return 0
	}
	c := 0
	for i, bt := range b {
		if i >= bitvector512ByteSize {
			break
		}
		c += bits.OnesCount8(bt)
	}
	return uint64(c)
}

func HashToByte(h common.Hash) []byte {
	ret := make([]byte, 0, len(h))
	for _, b := range h {
		ret = append(ret, b)
	}
	return ret
}
