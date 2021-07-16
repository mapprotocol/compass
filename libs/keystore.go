package libs

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
)

var privatekeyInKeystore *ecdsa.PrivateKey

func GetKey(password string) *ecdsa.PrivateKey {
	if privatekeyInKeystore != nil {
		return privatekeyInKeystore
	}
	path := ReadConfig("keystore", "keystore.json")
	//Compatible for development
	//You only need to deploy one keystore.json file at project root when you take the test
	if fileExist("../" + path) {
		path = "../" + path
	}
	if fileExist("../../" + path) {
		path = "../../" + path
	}
	if fileExist("../../../" + path) {
		path = "../../../" + path
	}
	for {
		if !fileExist(path) {
			print(path + " does not exist, please enter the keystore path: ")
			path = ReadString()
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
				log.Println("Password typed: " + string(password))
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
