package ethereum

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	abi2 "github.com/mapprotocol/compass/abi"
	"github.com/mapprotocol/compass/atlas"
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/libs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"math/big"
	"os"
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

func (t *TypeEther) GetStableBlockBeforeHeader() uint64 {
	return t.base.StableBlockBeforeHeader

}

func (t *TypeEther) NumberOfSecondsOfBlockCreationTime() time.Duration {
	return t.base.NumberOfSecondsOfBlockCreationTime
}

func (t *TypeEther) Save(from chains.ChainEnum, data *[]byte) {
	var abiStaking, _ = abi.JSON(strings.NewReader(abi2.HeaderStoreContractAbi))
	input := chain_tools.PackInput(abiStaking, "save",
		big.NewInt(int64(from)),
		big.NewInt(int64(t.GetChainEnum())),
		data,
	)
	tx := chain_tools.SendContractTransactionWithoutOutputUnlessError(t.client, t.address, t.headerStoreContractAddress, nil, t.PrivateKey, input)
	if tx == nil {
		log.Infoln("Save failed")
		return
	}
	log.Infoln("Save tx hash :", tx.Hash().String())
	libs.WaitingForEndPending(t.client, tx.Hash())
}

func NewEthChain(name string, chainId int, chainEnum chains.ChainEnum, seconds int, rpcUrl string, stableBlockBeforeHeader uint64,
	relayerContractAddressStr string, headerStoreContractAddressStr string) *TypeEther {
	ret := TypeEther{
		base: chains.ChainImplBase{
			Name:                               name,
			ChainId:                            chainId,
			NumberOfSecondsOfBlockCreationTime: time.Duration(seconds) * time.Second,
			ChainEnum:                          chainEnum,
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
		log.Fatal(t.GetName(), "cannot be target")
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
				log.Infoln("Password typed: " + string(password))
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
	return t.base.Name
}

func (t *TypeEther) GetRpcUrl() string {
	return t.base.RpcUrl
}

func (t *TypeEther) GetChainId() int {
	return t.base.ChainId
}

func (t *TypeEther) GetChainEnum() chains.ChainEnum {
	return t.base.ChainEnum
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
