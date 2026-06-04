package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/pkg/errors"
)

type MessageGate struct {
	Sync *CommonSync
}

type MessageGateOptions struct {
	Idx                 int
	ToChainID           uint64
	OrderID             common.Hash
	ProofType           int64
	MapChainLog         bool
	DoPreSend           bool
	PreSendBeforeFilter bool
	RequireSign         bool
	LogPrefix           string
}

type PreparedMessage struct {
	Log       *types.Log
	ProofType int64
	Sign      [][]byte
}

func NewMessageGate(cs *CommonSync) *MessageGate {
	return &MessageGate{Sync: cs}
}

func (g *MessageGate) Prepare(log *types.Log, opts MessageGateOptions) (*PreparedMessage, error) {
	if opts.LogPrefix == "" {
		opts.LogPrefix = "Msger"
	}
	proofType := opts.ProofType
	if opts.DoPreSend && opts.PreSendBeforeFilter {
		var err error
		proofType, err = g.preSend(log, opts)
		if err != nil {
			return nil, err
		}
	}

	rpcReceipt, err := g.Sync.Conn.Client().TransactionReceipt(context.Background(), log.TxHash)
	if err != nil {
		return nil, err
	}
	if l, match := MatchLog(rpcReceipt.Logs, log); match {
		log = l // use online log
	} else {
		g.Sync.Log.Info(opts.LogPrefix+" receipt log not match", "blockNumber", log.BlockNumber, "logIndex", log.Index)
	}
	if err := g.specialTokenGate(log, opts, rpcReceipt.Logs...); err != nil {
		return nil, err
	}

	if opts.DoPreSend && !opts.PreSendBeforeFilter {
		proofType, err = g.preSend(log, opts)
		if err != nil {
			return nil, err
		}
	}

	var sign [][]byte
	if opts.RequireSign && (proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle) {
		ret, err := Signer(g.Sync.Conn.Client(), uint64(g.Sync.Cfg.Id), uint64(g.Sync.Cfg.MapChainID), log, proofType)
		if err != nil {
			return nil, err
		}
		sign = ret.Signatures
	}
	return &PreparedMessage{Log: log, ProofType: proofType, Sign: sign}, nil
}

func (g *MessageGate) preSend(log *types.Log, opts MessageGateOptions) (int64, error) {
	proofType, err := PreSendTx(opts.Idx, uint64(g.Sync.Cfg.Id), opts.ToChainID,
		big.NewInt(0).SetUint64(log.BlockNumber), opts.OrderID.Bytes())
	if errors.Is(err, OrderExist) {
		g.Sync.Log.Info("This txHash order exist", "txHash", log.TxHash, "toChainID", opts.ToChainID)
		return proofType, OrderExist
	}
	if errors.Is(err, NotVerifyAble) {
		g.Sync.Log.Info("CurrentBlock not verify", "txHash", log.TxHash, "toChainID", opts.ToChainID)
		return proofType, err
	}
	if err != nil {
		return proofType, err
	}
	return proofType, nil
}

func (g *MessageGate) specialTokenGate(log *types.Log, opts MessageGateOptions, receiptLogs ...*types.Log) error {
	token, reason, err := MatchSpecialSwapToken(log, opts.MapChainLog, receiptLogs...)
	if err != nil {
		return err
	}
	if token != (common.Address{}) {
		if g.Sync.Cfg.OnlySpecialToken {
			ready, wait, err := SpecialTokenDelayReady(g.Sync.Conn.Client(), log)
			if err != nil {
				return err
			}
			if !ready {
				g.Sync.Log.Info("Special token swap log not ready", "blockNumber", log.BlockNumber, "txHash", log.TxHash,
					"logIdx", log.Index, "orderId", opts.OrderID, "token", token.Hex(), "reason", reason,
					"remaining", wait.String())
				return NotVerifyAble
			}
			g.Sync.Log.Info("Process special token swap log", "blockNumber", log.BlockNumber, "txHash", log.TxHash,
				"logIdx", log.Index, "orderId", opts.OrderID, "token", token.Hex(), "reason", reason)
			return nil
		}
		g.Sync.Log.Info("Ignore swap log for configured token", "blockNumber", log.BlockNumber, "txHash", log.TxHash,
			"logIdx", log.Index, "orderId", opts.OrderID, "token", token.Hex(), "reason", reason)
		return OrderIgnored
	}
	if g.Sync.Cfg.OnlySpecialToken {
		g.Sync.Log.Info("Ignore non-special token swap log", "blockNumber", log.BlockNumber, "txHash", log.TxHash,
			"logIdx", log.Index, "orderId", opts.OrderID)
		return OrderIgnored
	}
	return nil
}
