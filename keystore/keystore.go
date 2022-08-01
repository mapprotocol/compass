// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

/*
The keystore package is used to load keys from keystore files, both for live use and for testing.

The Keystore

The keystore file is used as a file representation of a key. It contains 4 parts:
- The key type (secp256k1, sr25519)
- The PublicKey
- The Address
- The ciphertext

This keystore also requires a password to decrypt into a usable key.
The keystore library can be used to both encrypt keys into keystores, and decrypt keystore into keys.
For more information on how to encrypt and decrypt from the command line, reference the README: https://github.com/ChainSafe/ChainBridge

The Keyring

The keyring provides predefined secp256k1 and srr25519 keys to use in testing.
These keys are automatically provided during runtime and stored in memory rather than being stored on disk.
There are 5 keys currenty supported: Alice, Bob, Charlie, Dave, and Eve.

*/
package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChainSafe/chainbridge-utils/crypto"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/key"
)

const EthChain = "ethereum"
const EnvPassword = "KEYSTORE_PASSWORD"

var keyMapping = map[string]string{
	"ethereum":  "secp256k1",
	"substrate": "sr25519",
}

// for cache the pswd for the same account
var keyPassCache = map[string][]byte{}

// KeypairFromAddress attempts to load the encrypted key file for the provided address,
// prompting the user for the password.
func KeypairFromAddress(addr, chainType, path string, insecure bool) (crypto.Keypair, error) {
	if insecure {
		//return insecureKeypairFromAddress(path, chainType)
		return nil, nil
	}
	path = fmt.Sprintf("%s/%s.key", path, addr)
	// Make sure key exists before prompting password
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file not found: %s", path)
	}

	var pswd []byte
	// find pswd in cache first;
	// if using one account for several chains, u dont need to input the pswd repetitive
	if cachepswd, exist := keyPassCache[addr]; exist {
		pswd = cachepswd
	} else {
		if pswdStr := os.Getenv(EnvPassword); pswdStr != "" {
			pswd = []byte(pswdStr)
		} else {
			pswd = GetPassword(fmt.Sprintf("Enter password for key %s:", path))
		}
		// cache inputed pswd
		keyPassCache[addr] = pswd
	}

	kp, err := ReadFromFileAndDecrypt(path, pswd, keyMapping[chainType])
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func NearKeyPairFrom(networkName, path string, id types.AccountID) (kp key.KeyPair, err error) {
	var creds struct {
		AccountID  types.AccountID     `json:"account_id"`
		PublicKey  key.Base58PublicKey `json:"public_key"`
		PrivateKey key.KeyPair         `json:"private_key"`
	}

	home := path
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return
		}
		fmt.Println("near keyPairFrom home is ", home)
	}

	credsFile := filepath.Join(home, ".near-credentials", networkName, fmt.Sprintf("%s.json", id))

	var cf *os.File
	if cf, err = os.Open(credsFile); err != nil {
		return
	}
	defer cf.Close()

	if err = json.NewDecoder(cf).Decode(&creds); err != nil {
		return
	}

	if creds.PublicKey.String() != creds.PrivateKey.PublicKey.String() {
		err = fmt.Errorf("inconsistent public key, %s != %s", creds.PublicKey.String(), creds.PrivateKey.PublicKey.String())
		return
	}
	kp = creds.PrivateKey

	return
}
