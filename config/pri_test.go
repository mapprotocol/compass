package config

import (
	"encoding/hex"
	"io/ioutil"
	"testing"

	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/atlas/accounts/keystore"
)

func Test_GetPrivateKeyFromFile(t *testing.T) {
	password := ""
	keyfile := "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-15T07-51-25.301943000Z--e0dc8d7f134d0a79019bef9c2fd4b2013a64fcd6"

	file, err := ioutil.ReadFile(keyfile)
	if err != nil {
		fmt.Printf("failed to read the keyfile at '%s': %v", keyfile, err)
	}
	key, err := keystore.DecryptKey(file, password)
	if err != nil {
		panic(fmt.Errorf("error decrypting key: %v", err))
	}

	private := key.PrivateKey
	fmt.Println("==============================  private key:", hex.EncodeToString(crypto.FromECDSA(private)))
	fmt.Println("==============================  public key:", common.Bytes2Hex(crypto.FromECDSAPub(&private.PublicKey)))
}
