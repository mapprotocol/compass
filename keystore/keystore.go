package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/key"
)

const (
	EnvPassword = "KEYSTORE_PASSWORD"
)

var pswCache = make(map[string][]byte)

func KeypairFromEth(path string) (*keystore.Key, error) {
	// Make sure key exists before prompting password
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file not found: %s", path)
	}

	var pswd = pswCache[path]
	if len(pswd) == 0 {
		pswd = GetPassword(fmt.Sprintf("Enter password for key %s:", path))
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keyFile failed, err:%s", err)
	}
	ret, err := keystore.DecryptKey(file, string(pswd))
	if err != nil {
		return nil, fmt.Errorf("DecryptKey failed, err:%s", err)
	}
	pswCache[path] = pswd

	return ret, nil
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
