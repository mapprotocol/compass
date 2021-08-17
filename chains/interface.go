package chains

import (
	"github.com/mapprotocol/compass/types"
	"math/big"
	"time"
)

type ChainId int

type ChainInterface interface {
	GetName() string
	GetChainId() ChainId
	GetBlockNumber() uint64
	GetRpcUrl() string
	GetBlockHeader(num uint64) *[]byte
	GetAddress() string
	SetTarget(keystoreStr string, password string)
	Save(from ChainId, Cdata *[]byte)
	NumberOfSecondsOfBlockCreationTime() time.Duration
	GetStableBlockBeforeHeader() uint64
	ContractInterface
}
type ChainImplBase struct {
	Name                               string
	ChainId                            ChainId
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
