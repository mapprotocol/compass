package constant

import (
	"errors"
	"time"
)

const (
	TxRetryInterval     = time.Second * 5 // TxRetryInterval Time between retrying a failed tx
	NearTxRetryInterval = time.Second * 20
)

var (
	ErrNonceTooLow  = errors.New("nonce too low")
	ErrUnWantedSync = errors.New("unwanted Sync")
)

var (
	BlockRetryInterval = time.Second * 5
	RetryLongInterval  = time.Second * 10
	QueryRetryInterval = time.Second * 30
)

var (
	MaintainerInterval = time.Second * 3
	MessengerInterval  = time.Second * 1
)

var (
	BalanceRetryInterval = time.Second * 60
)

var IgnoreError = map[string]struct{}{
	"order exist":                       {},
	"already known":                     {},
	"Header is have":                    {},
	"height error":                      {},
	"invalid start block":               {},
	"invalid syncing block":             {},
	"initialized or unknown epoch":      {},
	"no need to update exe headers":     {},
	"New block must have higher height": {},
	"the update finalized slot should be higher than the finalized slot":      {},
	"previous exe block headers should be updated before update light client": {},
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
