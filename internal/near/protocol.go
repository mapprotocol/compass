package near

import (
	"math/big"

	"github.com/mapprotocol/near-api-go/pkg/types"
)

var (
	NewFunctionCallGas types.Gas = 30 * 10000000000000
	Deposit                      = "0.3"
)

type Result struct {
	BlockHash   string        `json:"block_hash"`
	BlockHeight int           `json:"block_height"`
	Logs        []interface{} `json:"logs"`
	Result      []byte        `json:"result"`
}

type TransferOut struct {
	Token        string  `json:"token"`
	From         string  `json:"from"`
	OrderId      []byte  `json:"order_id"`
	FromChain    big.Int `json:"from_chain"`
	ToChain      big.Int `json:"to_chain"`
	To           []byte  `json:"to"`
	Amount       big.Int `json:"amount"`
	ToChainToken string  `json:"to_chain_token"`
}
