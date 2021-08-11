package chain_structs

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mapprotocol/compass/libs"
	"github.com/mapprotocol/compass/libs/contracts"
	contracts2 "github.com/mapprotocol/compass/libs/sync_libs/contracts"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

type TypeEther struct {
	base                       ChainImplBase
	client                     *ethclient.Client
	address                    common.Address    //if SetTarget is not called ,it's nil
	PrivateKey                 *ecdsa.PrivateKey //if SetTarget is not called ,it's nil
	relayerContractAddress     common.Address
	headerStoreContractAddress common.Address
}

func (t *TypeEther) GetStableBlockBeforeHeader() uint64 {
	return t.base.stableBlockBeforeHeader

}

func (t *TypeEther) NumberOfSecondsOfBlockCreationTime() time.Duration {
	return t.base.numberOfSecondsOfBlockCreationTime
}

func (t *TypeEther) SyncBlock(from ChainEnum, data *[]byte) {
	var abiStaking, _ = abi.JSON(strings.NewReader(contracts2.HeaderStoreContractAbi))
	input := contracts.PackInput(abiStaking, "save",
		big.NewInt(int64(from)),
		big.NewInt(int64(t.GetChainEnum())),
		data,
	)
	tx := contracts.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.headerStoreContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		log.Println("SyncBlock failed")
		return
	}
	log.Println("SyncBlock tx hash :", tx.Hash().String())
	libs.GetResult(t.client, tx.Hash())
}

func NewEthChain(name string, chainId int, chainEnum ChainEnum, seconds int, rpcUrl string, stableBlockBeforeHeader uint64,
	relayerContractAddressStr string, headerStoreContractAddressStr string) *TypeEther {
	ret := TypeEther{
		base: ChainImplBase{
			name:                               name,
			chainId:                            chainId,
			numberOfSecondsOfBlockCreationTime: time.Duration(seconds) * time.Second,
			chainEnum:                          chainEnum,
			rpcUrl:                             rpcUrl,
			stableBlockBeforeHeader:            stableBlockBeforeHeader,
		},
		client:                     libs.GetClientByUrl(rpcUrl),
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
		println(t.GetName(), "cannot be target")
		os.Exit(1)
	}
	keyJson, _ := ioutil.ReadFile(keystoreStr)
	var err error
	var key *keystore.Key
	if len(password) != 0 {
		key, err = keystore.DecryptKey(keyJson, password)
		if err != nil {
			println("Incorrect password! Modify the content in the .env file. It can be empty,but it can't be wrong.")
			os.Exit(1)
		}
	} else {
		for {
			print("Please enter your password: ")
			passwordByte, err := terminal.ReadPassword(0)
			if err != nil {
				log.Println("Password typed: " + string(password))
			}
			password = string(passwordByte)
			key, err = keystore.DecryptKey(keyJson, password)
			if err != nil {
				println("Incorrect password!")
			} else {
				println()
				break
			}
		}
	}
	t.PrivateKey = key.PrivateKey
	t.address = crypto.PubkeyToAddress(key.PrivateKey.PublicKey)

}

func (t *TypeEther) GetName() string {
	return t.base.name
}

func (t *TypeEther) GetRpcUrl() string {
	return t.base.rpcUrl
}

func (t *TypeEther) GetChainId() int {
	return t.base.chainId
}

func (t *TypeEther) GetChainEnum() ChainEnum {
	return t.base.chainEnum
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
	data, _ := json.Marshal([]*types.Header{block.Header()})
	return &data
}
