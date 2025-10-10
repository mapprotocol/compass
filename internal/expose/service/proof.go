package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/pkg/ethclient"

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
	pri *ecdsa.PrivateKey
}

func NewProof(cfg *expose.Config, pri *ecdsa.PrivateKey) *ProofSrv {
	return &ProofSrv{cfg: cfg, pri: pri}
}

func (s *ProofSrv) TxExec(req *stream.TxExecOfRequest) (map[string]interface{}, error) {
	switch req.Status {
	case constant.StatusOfRelayFailed:
		return s.RouterRetryMessageIn(s.cfg.Other.Butter, req.RelayChain, req.RelayTxHash)
	case constant.StatusOfSwapFailed, constant.StatusOfDesFailed:
		if req.Slippage == "" {
			req.Slippage = "100"
		}
		return s.RouterExecSwap(s.cfg.Other.Butter, req.DesChain, req.DesTxHash, req.Slippage)
	case constant.StatusOfInit:
		desChain := req.DesChain
		desChainInt, _ := strconv.ParseInt(desChain, 10, 64)
		srcChainInt, _ := strconv.ParseInt(req.SrcChain, 10, 64)
		if srcChainInt != constant.MapChainId && desChainInt != constant.MapChainId {
			desChain = strconv.FormatInt(constant.MapChainId, 10)
		}
		return s.SuccessProof(req.SrcChain, desChain, req.SrcBlockNumber, req.SrcLogIndex)
	case constant.StatusOfRelayFinish:
		return s.SuccessProof(req.RelayChain, req.DesChain, req.RelayBlockNumber, req.RelayLogIndex)
	default:
	}

	return nil, nil
}

func (s *ProofSrv) RouterExecSwap(butterHost, toChain, txHash, slippage string) (map[string]interface{}, error) {
	data, err := butter.ExecSwap(butterHost, fmt.Sprintf("toChainId=%s&txHash=%s&slippage=%s", toChain, txHash, slippage))
	if err != nil {
		return nil, err
	}

	var desTo string
	for _, ele := range s.cfg.Chains {
		if ele.Id == toChain {
			desTo = ele.Mcs
			break
		}
	}

	resp := butter.ExecSwapResp{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Errno != 0 {
		return nil, fmt.Errorf("swap failed with errno: %s", resp.Message)
	}

	return map[string]interface{}{
		"userRouter": true,
		"exec_chain": toChain,
		"exec_to":    desTo,
		"exec_data":  "0x",
		"exec_desc":  "failed tx retry exec",
		"exec_route": resp,
	}, nil
}

func (s *ProofSrv) RouterRetryMessageIn(butterHost, toChain, txHash string) (map[string]interface{}, error) {
	data, err := butter.RetryMessageIn(butterHost, fmt.Sprintf("txHash=%s", txHash))
	if err != nil {
		return nil, err
	}

	var desTo string
	for _, ele := range s.cfg.Chains {
		if ele.Id == toChain {
			desTo = ele.Mcs
			break
		}
	}

	resp := butter.RetryMessageInData{}
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Errno != 0 {
		return nil, fmt.Errorf("swap failed with errno: %s", resp.Message)
	}

	return map[string]interface{}{
		"userRouter": true,
		"exec_relay": true,
		"exec_chain": toChain,
		"exec_to":    desTo,
		"exec_data":  "0x",
		"exec_desc":  "exec failed relay tx retry",
		"exec_route": resp,
	}, nil
}

func (s *ProofSrv) SuccessProof(srcChain, desChain string, srcBlockNumber int64, logIndex uint) (map[string]interface{}, error) {
	var (
		err                                                                          error
		proofType                                                                    = int64(0)
		src, des                                                                     chains.Proffer
		srcClient                                                                    *ethclient.Client
		srcEndpoint, srcOracleNode, srcMcs, srcLightNode, desTo, desLight, desOracle string
		srcChainId, _                                                                = strconv.ParseUint(srcChain, 10, 64)
		desChainId, _                                                                = strconv.ParseUint(desChain, 10, 64)
	)
	for _, ele := range s.cfg.Chains {
		if ele.Id == srcChain {
			creator, _ := chains.CreateProffer(ele.Type)
			src = creator
			srcEndpoint = ele.Endpoint
			srcOracleNode = ele.OracleNode
			srcMcs = ele.Mcs
			srcLightNode = ele.LightNode
			srcClient, err = src.Connect(srcChain, srcEndpoint, srcMcs, srcLightNode, srcOracleNode)
			if err != nil {
				return nil, err
			}
		}
		if ele.Id == desChain {
			creator, _ := chains.CreateProffer(ele.Type)
			des = creator
			desTo = ele.Mcs
			desOracle = ele.OracleNode
			desLight = ele.LightNode
			_, err = des.Connect(desChain, ele.Endpoint, ele.Mcs, ele.LightNode, ele.OracleNode)
			if err != nil {
				return nil, err
			}
			if ele.Name == constant.Tron || ele.Name == constant.Ton || ele.Name == constant.Solana {
				proofType = constant.ProofTypeOfLogOracle
			}
		}
	}
	if src == nil {
		return nil, errors.New("srcChain unrecognized Chain Type")
	}

	if des == nil {
		return nil, errors.New("desChain unrecognized Chain Type")
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
		proofType, err = chain.PreSendTx(0, srcChainId, desChainId, big.NewInt(srcBlockNumber), orderId.Bytes())
		if errors.Is(err, chain.NotVerifyAble) { // maintainer
			updateHeader, err := src.Maintainer(srcClient, srcChainId, desChainId, srcEndpoint)
			if err != nil {
				return nil, errors.Wrap(err, "Assemble maintainer failed")
			}
			return map[string]interface{}{
				"userRouter": false,
				"exec_chain": desChain,
				"exec_to":    desLight,
				"exec_data":  "0x" + common.Bytes2Hex(updateHeader),
				"exec_desc":  "Execute maintainer transaction",
				"exec_route": struct{}{},
			}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	var sign [][]byte
	if proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle {
		ret, err := chain.Signer(srcClient, srcChainId, constant.MapChainId, &targetLog, proofType)
		if errors.Is(err, chain.NotVerifyAble) {
			oracle, err := chain.ExternalOracleInput(int64(srcChainId), proofType, &targetLog, srcClient, s.pri) // private
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"userRouter": false,
				"exec_chain": desChain,
				"exec_to":    desOracle,
				"exec_data":  "0x" + common.Bytes2Hex(oracle),
				"exec_desc":  "Execute oracle transaction",
				"exec_route": struct{}{},
			}, nil
		}
		if err != nil {
			return nil, err
		}
		sign = ret.Signatures
	}
	// proof
	ret, err := src.Proof(srcClient, &targetLog, srcEndpoint, proofType, srcChainId, desChainId, sign)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"userRouter": false,
		"exec_chain": desChain,
		"exec_to":    desTo,
		"exec_data":  "0x" + common.Bytes2Hex(ret),
		"exec_desc":  "Execute mos transaction",
		"exec_route": struct{}{},
	}, nil
}
