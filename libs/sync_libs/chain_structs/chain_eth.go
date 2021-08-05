package chain_structs

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"signmap/libs"
)

type TypeEther struct {
	name                       string
	chainEnum                  ChainEnum
	chainId                    int
	rpcUrl                     string
	client                     *ethclient.Client
	stableBlockBeforeHeader    int
	addressString              string            //if SetTarget is not called ,it's empty
	PrivateKey                 *ecdsa.PrivateKey //if SetTarget is not called ,it's nil
	relayerContractAddress     common.Address
	headerStoreContractAddress common.Address
}

func NewEthChain(name string, chainId int, chainEnum ChainEnum, rpcUrl string, stableBlockBeforeHeader int,
	relayerContractAddressStr string, headerStoreContractAddressStr string) *TypeEther {
	ret := TypeEther{
		name:                       name,
		chainId:                    chainId,
		chainEnum:                  chainEnum,
		rpcUrl:                     rpcUrl,
		client:                     libs.GetClientByUrl(rpcUrl),
		stableBlockBeforeHeader:    stableBlockBeforeHeader,
		relayerContractAddress:     common.HexToAddress(relayerContractAddressStr),
		headerStoreContractAddress: common.HexToAddress(headerStoreContractAddressStr),
	}
	return &ret
}

func (t *TypeEther) GetAddress() string {
	return t.addressString
}

func (t *TypeEther) SetTarget() {
	//todo
	panic("implement me")
}

func (t *TypeEther) GetName() string {
	return t.name
}

func (t *TypeEther) GetRpcUrl() string {
	return t.rpcUrl
}

func (t *TypeEther) GetChainId() int {
	return t.chainId
}

func (t *TypeEther) GetChainEnum() ChainEnum {
	return t.chainEnum
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
