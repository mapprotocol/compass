package libs

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"io/ioutil"
	"log"
)

var privatekeyInKeystore *ecdsa.PrivateKey

func GetKey(password string) *ecdsa.PrivateKey {
	if privatekeyInKeystore != nil {
		return privatekeyInKeystore
	}
	path := "keystore"
	//Compatible for development
	//You only need to deploy one keystore file at project root when you take the test
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
			print(path + "The default file does not exist, please enter the keystore address: ")
			path = readString()
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
			log.Fatal("Password mistake!")
		}
	} else {
		for {
			print("Please enter your password: ")
			password = readString()
			key, err1 = keystore.DecryptKey(keyJson, password)
			if err1 != nil {
				println("Password mistake!")
			} else {
				break
			}
		}
	}

	privatekeyInKeystore = key.PrivateKey
	return privatekeyInKeystore
}
