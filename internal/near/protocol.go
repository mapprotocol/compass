package near

import "github.com/mapprotocol/near-api-go/pkg/types"

var (
	NewFunctionCallGas types.Gas = 30 * 10000000000000
)

type Result struct {
	BlockHash   string        `json:"block_hash"`
	BlockHeight int           `json:"block_height"`
	Logs        []interface{} `json:"logs"`
	Result      []byte        `json:"result"`
}
