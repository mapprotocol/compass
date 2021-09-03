package chain_tools

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
	"io/ioutil"
	"os"
	"syscall"
)

func LoadPrivateKey(keystoreStr, password string) (key *keystore.Key, inputPassword string) {
	if !common.FileExist(keystoreStr) {
		log.Fatal("keystore file not exists.")
	}
	keyJson, _ := ioutil.ReadFile(keystoreStr)
	var err error
	if len(password) != 0 {
		key, err = keystore.DecryptKey(keyJson, password)
		if err != nil {
			log.Fatal("Incorrect password! Modify the content in the config file. It can be empty,but it can't be wrong. " +
				"use \"./map_rly password\" Generate an encrypted password set to config.toml ")
			os.Exit(1)
		}
	} else {
		for {
			print("Please enter your password: ")

			passwordByte, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Println("Password typed: " + string(password))
			}
			password = string(passwordByte)

			key, err = keystore.DecryptKey(keyJson, password)
			if err != nil {
				println("Incorrect password!")
			} else {
				println()
				inputPassword = password
				break
			}
		}
	}
	return
}
