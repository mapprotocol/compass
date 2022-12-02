package monitor

import "github.com/mapprotocol/compass/msg"

type Config struct {
	Name             string      // Human-readable chain name
	Id               msg.ChainId // ChainID
	Endpoint         string      // url for rpc endpoint
	Network          string      //
	From             string      // address of key to use
	KeystorePath     string      // Location of key files
	NearKeystorePath string      // Location of key files
	Insecure         bool        // Indicated whether the test keyring should be used
}
