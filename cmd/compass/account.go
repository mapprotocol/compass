// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"github.com/pkg/errors"
	"path/filepath"

	"github.com/lbtsm/gotron-sdk/pkg/account"

	log "github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/keystore"
	"github.com/urfave/cli/v2"
)

// dataHandler is a struct which wraps any extra data our CMD functions need that cannot be passed through parameters
type dataHandler struct {
	datadir string
}

// wrapHandler takes in a Cmd function (all declared below) and wraps
// it in the correct signature for the Cli Commands
func wrapHandler(hdl func(*cli.Context, *dataHandler) error) cli.ActionFunc {

	return func(ctx *cli.Context) error {
		err := startLogger(ctx)
		if err != nil {
			return err
		}

		datadir, err := getDataDir(ctx)
		if err != nil {
			return fmt.Errorf("failed to access the datadir: %w", err)
		}

		return hdl(ctx, &dataHandler{datadir: datadir})
	}
}

// handleImportCmd imports external keystores into the bridge
func handleImportCmd(ctx *cli.Context, dHandler *dataHandler) error {
	log.Info("Importing key...")

	if !ctx.Bool(config.TronFlag.Name) {
		return errors.New("only support tron")
	}
	var err error

	var password []byte = nil
	if pwdflag := ctx.String(config.PasswordFlag.Name); pwdflag != "" {
		password = []byte(pwdflag)
	}
	privkeyflag := ctx.String(config.PrivateKeyFlag.Name)
	if privkeyflag == "" {
		return fmt.Errorf("privateKey is nil")
	}
	if password == nil {
		password = keystore.GetPassword("Enter password to encrypt keystore file:")
	}

	name := ctx.String(config.TronKeyNameFlag.Name)
	keyName, err := account.ImportFromPrivateKey(privkeyflag, name, string(password))
	if err != nil {
		return fmt.Errorf("tron import private key failed, err is %v", err)
	}
	fmt.Println("tron keystore save, key is", keyName, " please save you config file")
	return nil
}

// getDataDir obtains the path to the keystore and returns it as a string
func getDataDir(ctx *cli.Context) (string, error) {
	// key directory is datadir/keystore/
	if dir := ctx.String(config.KeystorePathFlag.Name); dir != "" {
		datadir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		log.Trace(fmt.Sprintf("Using keystore dir: %s", datadir))
		return datadir, nil
	}
	return "", fmt.Errorf("datadir flag not supplied")
}
