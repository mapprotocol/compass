package constant

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

const (
	TxRetryInterval     = time.Second * 5 // TxRetryInterval Time between retrying a failed tx
	NearTxRetryInterval = time.Second * 30
)

var (
	ErrNonceTooLow  = errors.New("nonce too low")
	ErrUnWantedSync = errors.New("unwanted Sync")
)

var (
	BlockRetryInterval = time.Second * 5
	RetryLongInterval  = time.Second * 10
	QueryRetryInterval = time.Second * 10
)

var (
	MaintainerInterval = time.Second * 3
	MessengerInterval  = time.Second * 1
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
)

var (
	BalanceRetryInterval = time.Second * 60
)

var IgnoreError = map[string]struct{}{
	"order exist":                       {},
	"Header is have":                    {},
	"header is have":                    {},
	"height error":                      {},
	"invalid start block":               {},
	"invalid syncing block":             {},
	"initialized or unknown epoch":      {},
	"no need to update exe headers":     {},
	"could not replace existing tx":     {},
	"New block must have higher height": {},
	"round mismatch":                    {},
	"epoch mismatch":                    {},
	"headers size too big":              {},
	"Height error":                      {},
	"Update height0 error":              {},
	"invalid end exe header number":     {},
	"the update finalized slot should be higher than the finalized slot":      {},
	"previous exe block headers should be updated before update light client": {},
	"REVERT opcode executed":    {},
	"Validators repetition add": {},
	"oracle: already update":    {},
}

type BlockIdOfEth2 string

const (
	FinalBlockIdOfEth2   BlockIdOfEth2 = "finalized"
	HeadBlockIdOfEth2    BlockIdOfEth2 = "head"
	GenesisBlockIdOfEth2 BlockIdOfEth2 = "genesis"
)

const (
	SlotsPerEpoch   int64 = 32
	EpochsPerPeriod int64 = 256
)

const (
	ProofTypeOfOrigin = iota + 1
	ProofTypeOfZk
	ProofTypeOfOracle
)

const (
	LegacyTxType     = iota
	AccessListTxType = 0x01
	DynamicFeeTxType = 0x02
	BlobTxType       = 0x03
)

var (
	ReceiptStatusFailedRLP     = []byte{}
	ReceiptStatusSuccessfulRLP = []byte{0x01}
)

const (
	ReceiptStatusFailed = uint64(0)
)

const (
	MerlinChainId     = 4200
	OpChainId         = 10
	ArbChainId        = 421614
	ArbTestnetChainId = 42161
)

const (
	TopicsOfSwapInVerified = "0x71b6b465a3e1914ab78a5c4e72ed92c70071ccf1a1bdee55bc47174cbcd47605"
)
