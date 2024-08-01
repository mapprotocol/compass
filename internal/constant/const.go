package constant

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	TxRetryInterval      = time.Second * 5 // TxRetryInterval Time between retrying a failed tx
	ThirtySecondInterval = time.Second * 30
)

var (
	ErrNonceTooLow  = errors.New("nonce too low")
	ErrUnWantedSync = errors.New("unwanted Sync")
)

var (
	BlockRetryInterval = time.Second * 5
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
	"already verified":          {},
}

type BlockIdOfEth2 string

const (
	FinalBlockIdOfEth2 BlockIdOfEth2 = "finalized"
	HeadBlockIdOfEth2  BlockIdOfEth2 = "head"
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
	EthChainId        = 5
	ZkSyncChainId     = 324
	B2ChainId         = 223
	OpChainId         = 10
	BaseChainId       = 8453
	BlastChainId      = 81457
	ArbChainId        = 421614
	ArbTestnetChainId = 42161
	MantleChainId     = 5000
	ScrollChainId     = 534352
	ZkLinkChainId     = 810180
	DodoChainId       = 53457
	TronChainId       = 728126428
	NearChainId       = 1360100178526209
)

const (
	TopicsOfSwapInVerified = "0x71b6b465a3e1914ab78a5c4e72ed92c70071ccf1a1bdee55bc47174cbcd47605"
)

const (
	ReqInterval = int64(3)
)
const (
	ProjectOfMsger  = int64(1)
	ProjectOfOracle = int64(8)
)
const (
	FilterUrl      = "v1/mos/list"
	FilterBlockUrl = "v1/block"
)

var (
	MapLogIdx = make(map[string]int64)
)

const (
	FeeRentType = "fee.io"
)
