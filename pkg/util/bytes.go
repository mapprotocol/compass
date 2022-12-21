package util

import (
	"encoding/hex"
	"strings"
)

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
