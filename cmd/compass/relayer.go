// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mapprotocol/compass/chains/ethereum"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli/v2"
)

// handleRegisterCmd register an account as a relayer
func handleRegisterCmd(ctx *cli.Context) error {
	accountAddr := ctx.String(config.Account.Name)
	if accountAddr == "" {
		return errors.New("AN ACCOUNT MUST BE PROVIDED!")
	}

	value := ctx.Int64(config.Value.Name)
	if value < mapprotocol.RegisterAmount {
		return fmt.Errorf("AMOUNT MUST BIGGER than %d", mapprotocol.RegisterAmount)
	}

	log.Info("Register...", "Address", accountAddr, "Amount", value)

	// get map config
	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}

	// construct Map Chain object
	chainId, err := strconv.Atoi(cfg.MapChain.Id)
	if err != nil {
		return err
	}
	chainConfig := &core.ChainConfig{
		Name:           cfg.MapChain.Name,
		Id:             msg.ChainId(chainId),
		Endpoint:       cfg.MapChain.Endpoint,
		From:           accountAddr,
		KeystorePath:   cfg.KeystorePath,
		Insecure:       false,
		BlockstorePath: ctx.String(config.BlockstorePathFlag.Name),
		FreshStart:     ctx.Bool(config.FreshStartFlag.Name),
		LatestBlock:    ctx.Bool(config.LatestBlockFlag.Name),
		Opts:           cfg.MapChain.Opts,
	}
	var m *metrics.ChainMetrics

	logger := log.Root().New("register", accountAddr)
	sysErr := make(chan error)
	mapChain, err := ethereum.InitializeChain(chainConfig, logger, sysErr, m, ethereum.MarkOfMaintainer)
	if err != nil {
		return err
	}

	err = mapprotocol.RegisterRelayerWithConn(mapChain.Conn(), value, logger)
	if err != nil {
		return err
	}
	log.Info("Register sucessed...")

	return nil
}

// handleBindCmd register an account as a relayer
func handleBindCmd(ctx *cli.Context) error {
	relayerAddr := ctx.String(config.Relayer.Name)
	if relayerAddr == "" {
		return errors.New("RELAYER ADDRESS MUST BE PROVIDED!")
	}

	workerAddr := ctx.String(config.Worker.Name)
	if relayerAddr == "" {
		return errors.New("WORKER ADDRESS MUST BE PROVIDED!")
	}

	log.Info("Bind...", "Relayer", relayerAddr, "Worker", workerAddr)

	// get map config
	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}

	// construct Map Chain object
	chainId, err := strconv.Atoi(cfg.MapChain.Id)
	if err != nil {
		return err
	}
	chainConfig := &core.ChainConfig{
		Name:           cfg.MapChain.Name,
		Id:             msg.ChainId(chainId),
		Endpoint:       cfg.MapChain.Endpoint,
		From:           relayerAddr,
		KeystorePath:   cfg.KeystorePath,
		Insecure:       false,
		BlockstorePath: ctx.String(config.BlockstorePathFlag.Name),
		FreshStart:     ctx.Bool(config.FreshStartFlag.Name),
		LatestBlock:    ctx.Bool(config.LatestBlockFlag.Name),
		Opts:           cfg.MapChain.Opts,
	}
	var m *metrics.ChainMetrics

	logger := log.Root().New("bind", relayerAddr)
	sysErr := make(chan error)
	mapChain, err := ethereum.InitializeChain(chainConfig, logger, sysErr, m, ethereum.MarkOfMaintainer)
	if err != nil {
		return err
	}

	err = mapprotocol.BindWorkerWithConn(mapChain.Conn(), workerAddr, logger)
	if err != nil {
		return err
	}
	log.Info("Bind sucessed...")

	return nil
}
