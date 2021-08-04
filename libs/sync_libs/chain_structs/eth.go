package chain_structs

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"signmap/libs"
)

type TypeEther struct {
	chainId                    ChainId
	rpcUrl                     string
	client                     *ethclient.Client
	stableBlockBeforeHeader    int
	relayerContractAddress     common.Address
	headerStoreContractAddress common.Address
}

func NewEthChain(chainId ChainId, rpcUrl string, stableBlockBeforeHeader int,
	relayerContractAddressStr string, headerStoreContractAddressStr string) *TypeEther {
	return &TypeEther{
		chainId:                    chainId,
		rpcUrl:                     rpcUrl,
		client:                     libs.GetClientByUrl(rpcUrl),
		stableBlockBeforeHeader:    stableBlockBeforeHeader,
		relayerContractAddress:     common.HexToAddress(relayerContractAddressStr),
		headerStoreContractAddress: common.HexToAddress(headerStoreContractAddressStr),
	}
}

func (t *TypeEther) GetRpcUrl() string {
	return t.rpcUrl
}

func (t *TypeEther) GetChainId() ChainId {
	return t.chainId
}

func (t *TypeEther) GetBlockNumber() uint64 {
	num, err := t.client.BlockNumber(context.Background())
	if err != nil {
		return num
	}
	return 0
}

func (t *TypeEther) GetBlockHeader(num uint64) []byte {
	block, err := t.client.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return []byte{}
	}
	data, _ := block.Header().MarshalJSON()
	return data
}
