package chains

import (
	"github.com/mapprotocol/compass/types"
	"math/big"
	"time"
)

type ChainEnum int

type ChainInterface interface {
	GetName() string
	GetChainEnum() ChainEnum
	GetChainId() int
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) *[]byte
	GetAddress() string
	SetTarget(keystoreStr string, password string)
	Save(from ChainEnum, Cdata *[]byte)
	NumberOfSecondsOfBlockCreationTime() time.Duration
	GetStableBlockBeforeHeader() uint64
	ContractInterface
}
type ChainImplBase struct {
	Name                               string
	ChainEnum                          ChainEnum
	ChainId                            int
	RpcUrl                             string
	NumberOfSecondsOfBlockCreationTime time.Duration
	StableBlockBeforeHeader            uint64
}
type ContractInterface interface {
	Register(value *big.Int) bool
	UnRegister(value *big.Int) bool
	GetRelayerBalance() types.GetRelayerBalanceResponse
	GetRelayer() types.GetRelayerResponse
	GetPeriodHeight() types.GetPeriodHeightResponse
}

// relayContractAddressStr is empty,it cannot be target,
