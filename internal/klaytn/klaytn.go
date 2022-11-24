package klaytn

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Header struct {
	ParentHash       []byte         `json:"parentHash"`
	Reward           common.Address `json:"reward"`
	StateRoot        []byte         `json:"stateRoot"`
	TransactionsRoot []byte         `json:"transactionsRoot"`
	ReceiptsRoot     []byte         `json:"receiptsRoot"`
	LogsBloom        []byte         `json:"logsBloom"`
	BlockScore       *big.Int       `json:"blockScore"`
	Number           *big.Int       `json:"number"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        *big.Int       `json:"timestamp"`
	TimestampFoS     *big.Int       `json:"timestampFoS"`
	ExtraData        []byte         `json:"extraData"`
	GovernanceData   []byte         `json:"governanceData"`
	VoteData         []byte         `json:"voteData"`
	BaseFeePerGas    *big.Int       `json:"baseFeePerGas"`
}
