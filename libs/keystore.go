package libs

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
)

var privatekeyInKeystore *ecdsa.PrivateKey

func GetKey(password string) *ecdsa.PrivateKey {
	if privatekeyInKeystore != nil {
		return privatekeyInKeystore
	}
	path := ReadConfig("keystore", "keystore.json")
	//Compatible for development
	//You only need to deploy one keystore.json file at project root when you take the test

	if common.FileExist("../" + path) {
		path = "../" + path
	}
	if common.FileExist("../../" + path) {
		path = "../../" + path
	}
	if common.FileExist("../../../" + path) {
		path = "../../../" + path
	}
	for {
		if !common.FileExist(path) {
			print(path + " does not exist, please enter the keystore path: ")
			path = utils.ReadString()
		} else {
			break
		}
	}

	keyJson, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var err1 error
	var key *keystore.Key
	if len(password) != 0 {
		key, err1 = keystore.DecryptKey(keyJson, password)
		if err1 != nil {
			log.Fatal("Incorrect password!")
		}
	} else {
		for {
			print("Please enter your password: ")
			passwordByte, err := terminal.ReadPassword(0)
			if err != nil {
				log.Infoln("Password typed: " + string(password))
			}
			password = string(passwordByte)
			key, err1 = keystore.DecryptKey(keyJson, password)
			if err1 != nil {
				println("Incorrect password!")
			} else {
				println()
				break
			}
		}
	}

	privatekeyInKeystore = key.PrivateKey
	return privatekeyInKeystore
}
