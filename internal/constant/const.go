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
	BlockRetryInterval   = time.Second * 3
	QueryRetryInterval   = time.Second * 5
	MaintainerInterval   = time.Second * 3
	MessengerInterval    = time.Second * 1
	BalanceRetryInterval = time.Second * 60
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
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
	"0x6838b56d":                {}, //already_meet()
	"0x8bc9d07c":                {}, //already_proposal()
	"0x98087555":                {}, // order exist
	"already in use":            {}, // solana order exist
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
	ProofTypeOfNewOracle
	ProofTypeOfLogOracle
)

const (
	LegacyTxType     = iota
	AccessListTxType = 0x01
	DynamicFeeTxType = 0x02
	BlobTxType       = 0x03
	SetCodeTxType    = 0x04
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
	BscChainId        = 56
	MaticChainId      = 137
	MapChainId        = 22776
	CfxChainId        = 1030
	ZkSyncChainId     = 324
	OpChainId         = 10
	BaseChainId       = 8453
	BlastChainId      = 81457
	ArbChainId        = 421614
	ArbTestnetChainId = 42161
	MantleChainId     = 5000
	ScrollChainId     = 534352
	DodoChainId       = 53457
	TronChainId       = 728126428
	SolTestChainId    = 1360108768460811
	SolMainChainId    = 1360108768460801
	NearChainId       = 1360100178526209
	TonChainId        = 1360104473493506
	BtcChainId        = 1360095883558913
	XlayerId          = 196
	DoageChainId      = 1360121653362689
	LineaChainId      = 59144
)

const (
	ReqInterval = int64(3)
)
const (
	ProjectOfMsger  = int64(1)
	ProjectOfOther  = int64(7)
	ProjectOfOracle = int64(8)
)
const (
	FilterUrl       = "v1/mos/list"
	FilterBlockUrl  = "v1/block"
	FilterBtcLogUrl = "api/v1/logs"
)

const (
	FeeRentType = "fee.io"
)

const (
	Bsc      = "bsc"
	Conflux  = "conflux"
	Eth2     = "eth2"
	Ethereum = "ethereum"
	Klaytn   = "klaytn"
	Matic    = "matic"
	Near     = "near"
	Solana   = "sol"
	Ton      = "ton"
	Tron     = "tron"
	Btc      = "btc"
	Xrp      = "xrp"
)

// 0: init，2: relay failed，3: relay finish， 4: swap failed， 5: dst chain mos failed
const (
	StatusOfInit = iota
	StatusOfUnKnow
	StatusOfRelayFailed
	StatusOfRelayFinish
	StatusOfSwapFailed
	StatusOfDesFailed
)

const (
	ProofTypeOfContract = iota + 1
)

const (
	TokenLongAddressOfBtc  = "0x0000000000000000000000000000000000425443"
	TokenShortAddressOfBtc = "0x425443"
)
