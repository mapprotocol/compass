package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/mapprotocol/compass/internal/expose"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/urfave/cli/v2"
)

var exposeCommand = cli.Command{
	Name:        "expose",
	Usage:       "pprof expose",
	Description: "",
	Action:      api,
	Subcommands: []*cli.Command{},
	Flags:       append(app.Flags, cliFlags...),
}

func api(ctx *cli.Context) error {
	// logger
	err := startLogger(ctx)
	if err != nil {
		return err
	}
	log.Info("Starting Proof expose ...")
	// parse config
	cfg, err := expose.Local(ctx)
	if err != nil {
		return err
	}
	util.Init(cfg.Other.Env, cfg.Other.MonitorUrl)
	//
	e := expose.New(cfg)
	g := gin.New()
	g.POST("/failed/proof", e.FailedExec)
	g.POST("/new/proof", e.SuccessProof)
	err = g.Run(cfg.Other.Port)
	if err != nil {
		return err
	}

	return nil
}
