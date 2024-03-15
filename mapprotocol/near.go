package mapprotocol

import (
	"math/big"

	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/hash"
	"github.com/mapprotocol/near-api-go/pkg/types/key"
	"github.com/mapprotocol/near-api-go/pkg/types/signature"
)

const (
	TransferOut = "transfer out"
	DepositOut  = "deposit out"
)

type StreamerMessage struct {
	Block  client.BlockView `json:"block"`
	Shards []IndexerShard   `json:"shards"`
}

type IndexerShard struct {
	Chunk                    *IndexerChunkView                    `json:"chunk"`
	ReceiptExecutionOutcomes []IndexerExecutionOutcomeWithReceipt `json:"receipt_execution_outcomes"`
	ShardID                  types.ShardID                        `json:"shard_id"`
	StateChanges             StateChangesView                     `json:"state_changes"`
}

type IndexerChunkView struct {
	Author   types.AccountID        `json:"author"`
	Header   client.ChunkHeaderView `json:"header"`
	Receipts []ReceiptView          `json:"receipts"`
	//Transactions []IndexerTransactionWithOutcome `json:"transactions"`
}

type ReceiptView struct {
	PredecessorId types.AccountID `json:"predecessor_id"`
	ReceiverID    types.AccountID `json:"receiver_id"`
	ReceiptID     hash.CryptoHash `jsom:"receipt_id"`
	Receipt       Receipt         `json:"receipt"`
}

type Receipt struct {
	Action Action `json:"Action"`
}

type Action struct {
	Actions []interface{} `json:"actions"` //
	//Actions             []map[string]interface{} `json:"actions"` //
	GasPrice            string        `json:"gas_price"`
	InputDataIds        []interface{} `json:"input_data_ids"`
	OutputDataReceivers []interface{} `json:"output_data_receivers"`
	SignerID            string        `json:"signer_id"`
	SignerPublicKey     string        `json:"signer_public_key"`
}

type IndexerTransactionWithOutcome struct {
	Outcome     IndexerExecutionOutcomeWithOptionalReceipt `json:"outcome"`
	Transaction SignedTransactionView                      `json:"transaction"`
}

type SignedTransactionView struct {
	SignerID   types.AccountID           `json:"signer_id"`
	PublicKey  key.Base58PublicKey       `json:"public_key"`
	Nonce      types.Nonce               `json:"nonce"`
	ReceiverID types.AccountID           `json:"receiver_id"`
	Actions    []interface{}             `json:"actions"`
	Signature  signature.Base58Signature `json:"signature"`
	Hash       hash.CryptoHash           `json:"hash"`
}

type IndexerExecutionOutcomeWithReceipt struct {
	ExecutionOutcome ExecutionOutcomeWithIdView `json:"execution_outcome"`
	Receipt          ReceiptView                `json:"receipt"`
}

type ExecutionOutcomeWithIdView struct {
	BlockHash hash.CryptoHash      `json:"block_hash"`
	ID        hash.CryptoHash      `json:"id"`
	Outcome   ExecutionOutcomeView `json:"outcome"`
	Proof     MerklePath           `json:"proof"`
}

type ExecutionOutcomeView struct {
	ExecutorID  types.AccountID          `json:"executor_id"`
	GasBurnt    types.Gas                `json:"gas_burnt"`
	Logs        []string                 `json:"logs"`
	Metadata    Metadata                 `json:"metadata"`
	ReceiptIDs  []hash.CryptoHash        `json:"receipt_ids"`
	Status      client.TransactionStatus `json:"status"`
	TokensBurnt string                   `json:"tokens_burnt"` // "242953087248000000000"
}

type MerklePathItem struct {
	Hash      hash.CryptoHash `json:"hash"`
	Direction string          `json:"direction"`
}

type MerklePath = []MerklePathItem

type IndexerExecutionOutcomeWithOptionalReceipt struct {
	ExecutionOutcome ExecutionOutcomeWithIdView `json:"execution_outcome"`
	Receipt          *client.ReceiptView        `json:"receipt"`
}

type StateChangesView []StateChangeWithCauseView

type StateChangeWithCauseView struct {
	Type   TypeOfStateChange    `json:"type"`
	Cause  StateChangeCauseView `json:"cause"`
	Change StateChangeView      `json:"change"`
	// Value  StateChangeValueView `json:"value"`
}

type StateChangeCauseView struct {
	ReceiptHash string `json:"receipt_hash"`
	Type        string `json:"type"`
}

type StateChangeView struct {
	AccountId     types.AccountID `json:"account_id"`
	Amount        string          `json:"amount"`
	CodeHash      string          `json:"code_hash"`
	Locked        string          `json:"locked"`
	StoragePaidAt int64           `json:"storage_paid_at"`
	StorageUsage  int64           `json:"storage_usage"`
	CodeBase64    string          `json:"code_base_64"`
}

type AccessKey struct {
	Nonce      *big.Int
	Permission string
}

// type StateChangeValueView json.RawMessage

type (
	TypeOfStateChange string
)
