package ethereum

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	abi2 "github.com/mapprotocol/compass/abi"
	"github.com/mapprotocol/compass/atlas"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/chains"
	log "github.com/sirupsen/logrus"
	"math/big"
	"strings"
	"time"
)

type TypeEther struct {
	base                       chains.ChainImplBase
	client                     *ethclient.Client
	address                    common.Address    //if SetTarget is not called ,it's nil
	PrivateKey                 *ecdsa.PrivateKey //if SetTarget is not called ,it's nil
	relayerContractAddress     common.Address
	headerStoreContractAddress common.Address
}

func (t *TypeEther) GetClient() *ethclient.Client {
	return t.client
}

func (t *TypeEther) GetStableBlockBeforeHeader() uint64 {
	return t.base.StableBlockBeforeHeader

}

func (t *TypeEther) NumberOfSecondsOfBlockCreationTime() time.Duration {
	return t.base.NumberOfSecondsOfBlockCreationTime
}

func (t *TypeEther) Save(from chains.ChainId, data *[]byte) {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.HeaderStoreContractAbi))
	input := chain_tools.PackInput(abiStaking, "save",
		big.NewInt(int64(from)),
		big.NewInt(int64(t.GetChainId())),
		data,
	)
	tx := chain_tools.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.headerStoreContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		log.Infoln("Save failed")
		return
	}
	log.Infoln("Save tx hash :", tx.Hash().String())
	chain_tools.WaitingForEndPending(t.client, tx.Hash(), 50)
}

func NewEthChain(name string, chainId chains.ChainId, seconds int, rpcUrl string, stableBlockBeforeHeader uint64,
	relayerContractAddressStr string, headerStoreContractAddressStr string) *TypeEther {
	ret := TypeEther{
		base: chains.ChainImplBase{
			Name:                               name,
			ChainId:                            chainId,
			NumberOfSecondsOfBlockCreationTime: time.Duration(seconds) * time.Second,
			RpcUrl:                             rpcUrl,
			StableBlockBeforeHeader:            stableBlockBeforeHeader,
		},
		client:                     chain_tools.GetClientByUrl(rpcUrl),
		relayerContractAddress:     common.HexToAddress(relayerContractAddressStr),
		headerStoreContractAddress: common.HexToAddress(headerStoreContractAddressStr),
	}
	return &ret
}

func (t *TypeEther) GetAddress() string {
	return t.address.String()
}

func (t *TypeEther) SetTarget(keystoreStr string, password string) {
	if t.relayerContractAddress.String() == "0x0000000000000000000000000000000000000000" ||
		t.headerStoreContractAddress.String() == "0x0000000000000000000000000000000000000000" {
		log.Fatal(t.GetName(), " cannot be target, relayer_contract_address and header_store_contract_address are required for target.")
	}
	key, _ := chain_tools.LoadPrivateKey(keystoreStr, password)
	t.PrivateKey = key.PrivateKey
	t.address = crypto.PubkeyToAddress(key.PrivateKey.PublicKey)

}

func (t *TypeEther) GetName() string {
	return t.base.Name
}

func (t *TypeEther) GetRpcUrl() string {
	return t.base.RpcUrl
}

func (t *TypeEther) GetChainId() chains.ChainId {
	return t.base.ChainId
}

func (t *TypeEther) GetBlockNumber() uint64 {
	num, err := t.client.BlockNumber(context.Background())
	if err == nil {
		return num
	}
	return 0
}

func (t *TypeEther) GetBlockHeader(num uint64) *[]byte {
	block, err := t.client.BlockByNumber(context.Background(), big.NewInt(int64(num)))
	if err != nil {
		return &[]byte{}
	}
	data, _ := json.Marshal([]*atlas.Header{chain_tools.ConvertHeader(block.Header())})
	return &data
}
