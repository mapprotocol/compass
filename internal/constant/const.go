package constant

import (
	"errors"
	"math/big"
	"time"
)

const (
	TxRetryInterval = time.Second * 3 // TxRetryInterval Time between retrying a failed tx
	TxRetryLimit    = 10              // TxRetryLimit Maximum number of tx retries before exiting
	HttpTimeOut     = 10 * time.Second
	Agent           = "compass-go"
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
	ErrFatalTx       = errors.New("submission of transaction failed")
)

var (
	BlockRetryLimit    = 20
	BlockRetryInterval = time.Second * 5
	RetryLongInterval  = time.Second * 10
	ErrFatalPolling    = errors.New("listener block polling failed")
)

var (
	BalanceRetryInterval = time.Second * 60
	Wei                  = new(big.Int).SetUint64(1000000000)
	WeiOfNear, _         = new(big.Int).SetString("1000000000000000000000000", 10)
)

var (
	NotEnoughGas                 = "insufficient funds for gas * price + value"
	NotEnoughGasPrint            = "insufficient funds for gas * price + value, will retry"
	EthOrderExist                = "order exist"
	EthOrderExistPrint           = "Order Exist, Continue to the next"
	HeaderIsHave                 = "Header is have"
	HeaderIsHavePrint            = "Header is have, Continue to the next"
	InvalidStartBlock            = "invalid start block"
	InvalidStartBlockPrint       = "invalid start block, Continue to the next"
	InvalidSyncBlock             = "invalid syncing block"
	InvalidSyncBlockPrint        = "invalid syncing block, Continue to the next"
	NotPerMission                = "mosRelay :: only admin"
	NotPerMissionPrint           = "mosRelay :: only admin, will retry"
	AddressIsZero                = "address is zero"
	AddressIsZeroPrint           = "address is zero, will retry"
	VaultNotRegister             = "vault token not registered"
	VaultNotRegisterPrint        = "vault token not registered, will retry"
	InvalidVaultToken            = "Invalid vault token"
	InvalidVaultTokenPrint       = "Invalid vault token, will retry"
	InvalidMosContract           = "invalid mos contract"
	InvalidMosContractPrint      = "invalid mos contract, will retry"
	InvalidChainId               = "invalid chain id"
	InvalidChainIdPrint          = "invalid chain id, will retry"
	MapTokenNotRegistered        = "map token not registered"
	MapTokenNotRegisteredPrint   = "map token not registered, will retry"
	OutTokenNotRegistered        = "out token not registered"
	OutTokenNotRegisteredPrint   = "out token not registered, will retry"
	BalanceTooLow                = "balance too low"
	BalanceTooLowPrint           = "balance too low, will retry"
	VaultTokenNotRegistered      = "vault token not registered"
	VaultTokenNotRegisteredPrint = "vault token not registered, will retry"
	ChainTypeError               = "chain type error"
	ChainTypeErrorPrint          = "chain type error, will retry"
)

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
