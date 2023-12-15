package tron

import (
	"fmt"
	"os"

	tronks "github.com/lbtsm/gotron-sdk/pkg/keystore"
	"github.com/lbtsm/gotron-sdk/pkg/store"
	"github.com/mapprotocol/compass/keystore"
)

func getKsAndAcc(from string) (*tronks.KeyStore, *tronks.Account, error) {
	var pswd []byte
	if pswdStr := os.Getenv(keystore.EnvPassword); pswdStr != "" {
		pswd = []byte(pswdStr)
	} else {
		pswd = keystore.GetPassword(fmt.Sprintf("Enter password for key %s:", from))
	}

	// getKeystore
	return store.UnlockedKeystore(from, string(pswd))
}
