package chain_tools

import "github.com/ethereum/go-ethereum/crypto"

var (
	Event1Key      = "Event1Key"
	Event1ArrayKey = "event1ArrayKey"
	Event1Hash     = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
)
