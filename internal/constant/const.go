package constant

import (
	"errors"
	"time"
)

const (
	TxRetryInterval = time.Second * 3 // TxRetryInterval Time between retrying a failed tx
	TxRetryLimit    = 10              // TxRetryLimit Maximum number of tx retries before exiting
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
	ErrFatalTx       = errors.New("submission of transaction failed")
)

var (
	PassedStatus      uint8 = 2
	TransferredStatus uint8 = 3
	CancelledStatus   uint8 = 4
)

var (
	BlockRetryLimit    = 5
	BlockRetryInterval = time.Second * 5
	ErrFatalPolling    = errors.New("listener block polling failed")
)

var (
	NotEnoughGas      = "insufficient funds for gas * price + value"
	NotEnoughGasPrint = "insufficient funds for gas * price + value, will retry"
	EthOrderExist     = "order exist"
)
