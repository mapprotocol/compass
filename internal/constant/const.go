package constant

import (
	"errors"
	"math/big"
	"time"
)

const (
	TxRetryInterval = time.Second * 5 // TxRetryInterval Time between retrying a failed tx
	TxRetryLimit    = 10              // TxRetryLimit Maximum number of tx retries before exiting
	HttpTimeOut     = 10 * time.Second
	Agent           = "compass-go"
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
	ErrUnWantedSync  = errors.New("unwanted Sync")
)

var (
	BlockRetryLimit    = 20
	BlockRetryInterval = time.Second * 5
	RetryLongInterval  = time.Second * 10
	QueryRetryInterval = time.Second * 30
	ErrFatalPolling    = errors.New("listener block polling failed")
)

var (
	BalanceRetryInterval = time.Second * 60
	WeiOfNear, _         = new(big.Int).SetString("1000000000000000000000000", 10)
)

var IgnoreError = map[string]struct{}{
	"order exist":                       {},
	"Header is have":                    {},
	"invalid start block":               {},
	"invalid syncing block":             {},
	"initialized or unknown epoch":      {},
	"New block must have higher height": {},
	"the update finalized slot should be higher than the finalized slot": {},
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
