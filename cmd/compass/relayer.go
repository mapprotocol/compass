// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"github.com/mapprotocol/compass/config"

	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli/v2"
)

// handleRegisterCmd register an account as a relayer
func handleRegisterCmd(ctx *cli.Context) error {
	log.Info("Register Account...")

	// get map config
	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}
	log.Info(cfg.KeystorePath)

	return nil
}

// getDataDir obtains the path to the keystore and returns it as a string
// func getDataDir(ctx *cli.Context) (string, error) {
// 	// key directory is datadir/keystore/
// 	if dir := ctx.String(config.KeystorePathFlag.Name); dir != "" {
// 		datadir, err := filepath.Abs(dir)
// 		if err != nil {
// 			return "", err
// 		}
// 		log.Trace(fmt.Sprintf("Using keystore dir: %s", datadir))
// 		return datadir, nil
// 	}
// 	return "", fmt.Errorf("datadir flag not supplied")
// }
