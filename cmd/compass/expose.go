package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/mapprotocol/compass/chains"
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
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
	// pre init
	for _, ele := range cfg.Chains {
		creator, _ := chains.CreateProffer(ele.Type)
		_, err = creator.Connect(ele.Id, ele.Endpoint, ele.Mcs, ele.OracleNode)
		if err != nil {
			return err
		}
	}

	e := expose.New(cfg)
	g := gin.New()
	g.Use(CORSMiddleware())
	g.POST("/failed/proof", e.FailedExec)
	g.POST("/new/proof", e.SuccessProof)
	err = g.Run(cfg.Other.Port)
	if err != nil {
		return err
	}

	return nil
}
