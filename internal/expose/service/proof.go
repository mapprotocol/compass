package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strconv"

	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/expose"
	"github.com/mapprotocol/compass/internal/stream"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

type ProofSrv struct {
	cfg *expose.Config
}

func NewProof(cfg *expose.Config) *ProofSrv {
	return &ProofSrv{cfg: cfg}
}

func (s *ProofSrv) TxExec(req *stream.TxExecOfRequest) (map[string]interface{}, error) {
	switch req.Status {
	case constant.StatusOfRelayFailed:
		return s.RouterExecSwap(s.cfg.Other.Butter, req.RelayChain, req.RelayTxHash)
	case constant.StatusOfSwapFailed, constant.StatusOfDesFailed:
		return s.RouterExecSwap(s.cfg.Other.Butter, req.DesChain, req.DesTxHash)
	case constant.StatusOfInit:
		return s.SuccessProof(req.SrcChain, req.RelayChain, req.SrcBlockNumber, req.SrcLogIndex)
	case constant.StatusOfRelayFinish:
		return s.SuccessProof(req.RelayChain, req.DesChain, req.RelayBlockNumber, req.RelayLogIndex)
	default:
	}

	return nil, nil
}

func (s *ProofSrv) RouterExecSwap(butterHost, toChain, txHash string) (map[string]interface{}, error) {
	data, err := butter.ExecSwap(butterHost, fmt.Sprintf("toChainId=%s&txHash=%s", toChain, txHash))
	if err != nil {
		return nil, err
	}

	resp := butter.ExecSwapResp{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Errno != 0 {
		return nil, fmt.Errorf("swap failed with errno: %d", resp.Message)
	}

	return map[string]interface{}{
		"userRouter": true,
		"exec_chain": toChain,
		"exec_to":    txHash,
		"exec_data":  "0x",
		"exec_descp": "failed tx retry exec",
		"exec_route": resp,
	}, nil
}

func (s *ProofSrv) SuccessProof(srcChain, desChain string, srcBlockNumber int64, logIndex uint) (map[string]interface{}, error) {

	var (
		proofType                                 = int64(0)
		src, des                                  chains.Proffer
		srcEndpoint, srcOracleNode, srcMcs, desTo string
		selfChainId, _                            = strconv.ParseUint(srcChain, 10, 64)
		desChainId, _                             = strconv.ParseUint(desChain, 10, 64)
	)
	for _, ele := range s.cfg.Chains {
		if ele.Id == srcChain {
			creator, _ := chains.CreateProffer(ele.Type)
			src = creator
			srcEndpoint = ele.Endpoint
			srcOracleNode = ele.OracleNode
			srcMcs = ele.Mcs
		}
		if ele.Id == desChain {
			creator, _ := chains.CreateProffer(ele.Type)
			des = creator
			desTo = ele.Mcs
			if ele.Name == constant.Tron || ele.Name == constant.Ton || ele.Name == constant.Solana {
				proofType = constant.ProofTypeOfLogOracle
			}
		}
	}
	if src == nil || des == nil {
		return nil, errors.New("srcChain or desChain unrecognized Chain Type")
	}
	srcClient, err := src.Connect(srcChain, srcEndpoint, srcMcs, srcOracleNode)
	if err != nil {
		return nil, err
	}
	// get log
	logs, err := srcClient.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(srcBlockNumber),
		ToBlock:   big.NewInt(srcBlockNumber),
		Addresses: nil,
		Topics:    nil,
	})
	if err != nil {
		return nil, err
	}
	targetLog := types.Log{}
	for _, ele := range logs {
		if ele.Index != logIndex {
			continue
		}
		tmp := ele
		targetLog = tmp
	}
	if proofType == 0 {
		orderId := targetLog.Topics[1]
		proofType, err = chain.PreSendTx(0, selfChainId, desChainId, big.NewInt(srcBlockNumber), orderId.Bytes())
		if err != nil {
			return nil, err
		}
	}
	var sign [][]byte
	if proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle {
		ret, err := chain.Signer(srcClient, selfChainId, 22776, &targetLog, proofType)
		if err != nil {
			return nil, err
		}
		sign = ret.Signatures
	}
	// proof
	ret, err := src.Proof(srcClient, &targetLog, srcEndpoint, proofType, selfChainId, desChainId, sign)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"userRouter": false,
		"exec_chain": desChain,
		"exec_to":    desTo,
		"exec_data":  "0x" + common.Bytes2Hex(ret),
		"exec_descp": "execute transaction",
		"exec_route": nil,
	}, nil
}
