package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gin-gonic/gin"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/expose"
	"github.com/mapprotocol/compass/internal/expose/service"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/pkg/errors"
	"math/big"
	"net/http"
	"strconv"
)

type Expose struct {
	cfg      *expose.Config
	proofSrv *service.ProofSrv
}

func New(cfg *expose.Config) *Expose {
	return &Expose{proofSrv: service.NewProof(cfg), cfg: cfg}
}

func (e *Expose) TxExec(c *gin.Context) {
	var req stream.TxExecOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	ret, err := e.proofSrv.TxExec(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	c.JSON(http.StatusOK, Success(map[string]interface{}{
		"data": ret,
	}))
}

func (e *Expose) FailedExec(c *gin.Context) {
	var req stream.FailedTxOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	data, err := butter.ExecSwap(e.cfg.Other.Butter, fmt.Sprintf("toChainId=%s&txHash=%s", req.ToChain, req.Hash))
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}

	ret := make(map[string]interface{})
	err = json.Unmarshal(data, &ret)
	if err != nil {
		c.JSON(http.StatusOK, Error2Response(err))
		return
	}

	c.JSON(http.StatusOK, ret)
}

func (e *Expose) SuccessProof(c *gin.Context) {
	var req stream.ProofOfRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}
	// init chain
	var (
		proofType                                 = int64(0)
		src, des                                  chains.Proffer
		srcEndpoint, srcOracleNode, srcMcs, desTo string
		selfChainId, _                            = strconv.ParseUint(req.SrcChain, 10, 64)
		desChainId, _                             = strconv.ParseUint(req.DesChain, 10, 64)
	)
	for _, ele := range e.cfg.Chains {
		if ele.Id == req.SrcChain {
			creator, _ := chains.CreateProffer(ele.Type)
			src = creator
			srcEndpoint = ele.Endpoint
			srcOracleNode = ele.OracleNode
			srcMcs = ele.Mcs
		}
		if ele.Id == req.DesChain {
			creator, _ := chains.CreateProffer(ele.Type)
			des = creator
			desTo = ele.Mcs
			if ele.Name == constant.Tron || ele.Name == constant.Ton || ele.Name == constant.Solana {
				proofType = constant.ProofTypeOfLogOracle
			}
		}
	}
	if src == nil || des == nil {
		log.Info("no proof of chain exists", "src", src, "des", des)
		c.JSON(http.StatusBadRequest, Error2Response(errors.New("srcChain or desChain unrecognized Chain Type")))
		return
	}
	srcClient, err := src.Connect(req.SrcChain, srcEndpoint, srcMcs, srcOracleNode)
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}
	// get log
	logs, err := srcClient.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(req.BlockNumber),
		ToBlock:   big.NewInt(req.BlockNumber),
		Addresses: nil,
		Topics:    nil,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}
	targetLog := types.Log{}
	for _, ele := range logs {
		if ele.Index != req.SrcLogIndex {
			continue
		}
		tmp := ele
		targetLog = tmp
	}
	if proofType == 0 {
		orderId := targetLog.Topics[1]
		proofType, err = chain.PreSendTx(0, selfChainId, desChainId, big.NewInt(req.BlockNumber), orderId.Bytes())
		if err != nil {
			c.JSON(http.StatusBadRequest, Error2Response(err))
			return
		}
	}
	var sign [][]byte
	if proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle {
		ret, err := chain.Signer(srcClient, selfChainId, 22776, &targetLog, proofType)
		if err != nil {
			c.JSON(http.StatusBadRequest, Error2Response(err))
			return
		}
		sign = ret.Signatures
	}
	// proof
	ret, err := src.Proof(srcClient, &targetLog, srcEndpoint, proofType, selfChainId, desChainId, sign)
	if err != nil {
		c.JSON(http.StatusBadRequest, Error2Response(err))
		return
	}
	// back
	c.JSON(http.StatusOK, map[string]interface{}{
		"tx": "0x" + common.Bytes2Hex(ret),
		"to": desTo,
	})
}

func Error2Response(err error) interface{} {
	return map[string]interface{}{
		"code": 500,
		"msg":  err.Error(),
	}
}

func Success(data interface{}) interface{} {
	return map[string]interface{}{
		"code": 0,
		"msg":  "success",
		"data": data,
	}
}
