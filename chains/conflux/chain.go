package conflux

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/mapprotocol/compass/internal/conflux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/bsc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

var (
	skippedRound uint64
	startNumber  uint64 = 130984900
	cli                 = &conflux.Client{}
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	client, err := conflux.NewClient(chainCfg.Opts[chain.Eth2Url])
	if err != nil {
		panic("conflux init client failed" + err.Error())
	}
	cli = client
	return chain.New(chainCfg, logger, sysErr, m, role, connection.NewConnection,
		chain.OptOfSync2Map(syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderCountOfConflux),
		chain.OptOfMos(mosHandler),
	)
}

func syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
	state, err := getState(m)
	if err != nil {
		m.Log.Error("Conflux GetConfluxState Failed", "err", err)
		return err
	}
	m.Log.Info("Conflux GetState", "state", state)
	epoch := state.Epoch.Uint64()
	round := state.Round.Uint64() + 1
	if skippedRound > 0 {
		round = skippedRound + 1
	}
	if startNumber == 0 {
		startNumber = state.FinalizedBlockNumber.Uint64()
	} else if state.FinalizedBlockNumber.Uint64() > startNumber {
		err = updateHeaders(m, startNumber, state.FinalizedBlockNumber.Uint64())
		if err != nil {
			return err
		}
		startNumber = state.FinalizedBlockNumber.Uint64()
	}

	committed, err := isCommitted(epoch, round)
	if err != nil {
		m.Log.Error("Conflux isCommitted Failed", "err", err)
		return err
	}
	m.Log.Info("Conflux isCommitted", "committed", committed)
	if !committed {
		logrus.WithField("epoch", epoch).WithField("round", round).Debug("No pos block to relay")
		return nil
	}

	ledger, err := cli.GetLedgerInfoByEpochAndRound(
		context.Background(),
		hexutil.Uint64(epoch),
		hexutil.Uint64(round),
	)
	if err != nil {
		return errors.WithMessage(err, "Failed to get ledger by epoch and round")
	}

	// no ledger in round, just skip it
	if ledger == nil {
		m.Log.Info("No ledger info in this round", "epoch", epoch)
		return nil
	}

	pivot := ledger.LedgerInfo.CommitInfo.Pivot

	// both committee and pow pivot block unchanged
	if ledger.LedgerInfo.CommitInfo.NextEpochState == nil {
		if pivot == nil || uint64(pivot.Height) <= state.FinalizedBlockNumber.Uint64() {
			m.Log.Info("Pos block pivot not changed", "pivot", pivot,
				"finalizedBlockNumber", state.FinalizedBlockNumber, "epoch", epoch, "round", round)
			skippedRound = round
			return nil
		}
	}

	input, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(conflux.ConvertLedger(ledger))
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	//fmt.Println("input --------- ", "0x"+ethcommon.Bytes2Hex(input))
	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, input, true}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return err
	}

	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	skippedRound = 0
	return nil
}

func mosHandler(m *chain.Messenger, latestBlock *big.Int) (int, error) {
	if !m.Cfg.SyncToMap {
		return 0, nil
	}
	m.Log.Debug("Querying block for events", "block", latestBlock)
	query := m.BuildQuery(m.Cfg.McsContract, m.Cfg.Events, latestBlock, latestBlock)
	// querying for logs
	logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
	if err != nil {
		return 0, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	m.Log.Debug("event", "latestBlock ", latestBlock, " logs ", len(logs))
	count := 0
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		// evm event to msg
		var message msg.Message
		// getOrderId
		orderId := log.Data[:32]
		method := m.GetMethod(log.Topics[0])
		// when syncToMap we need to assemble a tx proof
		txsHash, err := tx.GetTxsHashByBlockNumber(m.Conn.Client(), latestBlock)
		if err != nil {
			return 0, fmt.Errorf("unable to get tx hashes Logs: %w", err)
		}
		receipts, err := tx.GetReceiptsByTxsHash(m.Conn.Client(), txsHash)
		if err != nil {
			return 0, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
		}

		headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
		for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
			headerHeight := new(big.Int).Add(latestBlock, new(big.Int).SetInt64(int64(i)))
			header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
			if err != nil {
				return 0, err
			}
			headers[i] = *header
		}

		params := make([]bsc.Header, 0, len(headers))
		for _, h := range headers {
			params = append(params, bsc.ConvertHeader(h))
		}

		payload, err := bsc.AssembleProof(params, log, receipts, method, m.Cfg.Id)
		if err != nil {
			return 0, fmt.Errorf("unable to Parse Log: %w", err)
		}

		msgPayload := []interface{}{payload, orderId, latestBlock.Uint64(), log.TxHash}
		message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)

		m.Log.Info("Event found", "BlockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index,
			"orderId", ethcommon.Bytes2Hex(orderId))
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("Subscription error: failed to route message", "err", err)
		}
		count++
	}

	return count, nil
}

func getState(m *chain.Maintainer) (*conflux.ILightNodeState, error) {
	data, err := mapprotocol.GetManagerState(m.Cfg.Id)
	if err != nil {
		return nil, err
	}
	analysis, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfHeaderState].Outputs.Unpack(data)
	if err != nil {
		return nil, errors.Wrap(err, "analysis")
	}
	ret := new(conflux.ILightNodeState)
	if err = mapprotocol.Conflux.Methods[mapprotocol.MethodOfHeaderState].Outputs.Copy(&ret, analysis); err != nil {
		return nil, errors.Wrap(err, "analysis copy")
	}
	return ret, nil
}

func updateHeaders(m *chain.Maintainer, startNumber, endNumber uint64) error {
	m.Log.Info("Sync Header", "startNumber", startNumber, "endNumber", endNumber)
	headers := make([][]byte, mapprotocol.HeaderLengthOfConflux)
	idx := mapprotocol.HeaderLengthOfConflux - 1
	endNumber = 130984920
	startNumber = 130984901
	for i := startNumber; i <= endNumber; i++ {
		blk, err := cli.GetBlockByBlockNumber(context.Background(), hexutil.Uint64(i))
		if err != nil {
			return err
		}

		m.Log.Info("updateHeaders ", "height", i, "idx", idx)
		ele := conflux.MustRLPEncodeBlock(blk)
		headers[idx] = ele
		//idx++
		idx--
		if idx != -1 && i != endNumber {
			continue
		}
		if i == endNumber {
			headers = headers[idx+1:]
		}

		input, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfGetBlockHeadersBytes].Inputs.Pack(headers)
		if err != nil {
			m.Log.Error("Failed to header abi pack", "err", err)
			return err
		}
		fmt.Println("input ", "0x"+ethcommon.Bytes2Hex(input))
		id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
		msgPayload := []interface{}{id, input}
		message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
		err = m.Router.Send(message)
		if err != nil {
			m.Log.Error("Subscription header error: failed to route message", "err", err)
			return nil
		}
		err = m.WaitUntilMsgHandled(1)
		if err != nil {
			return err
		}
		idx = mapprotocol.HeaderLengthOfConflux - 1
		time.Sleep(time.Second * 2)
	}

	return nil
}

func isCommitted(epoch, round uint64) (bool, error) {
	status, err := cli.GetStatus(context.Background())
	if err != nil {
		return false, errors.WithMessage(err, "Failed to get pos status")
	}

	block, err := cli.GetBlockByNumber(context.Background(), conflux.NewBlockNumber(status.LatestCommitted))
	if err != nil {
		return false, errors.WithMessage(err, "Failed to get the latest committed block")
	}

	if block == nil {
		logrus.Fatal("Latest committed PoS block is nil")
	}

	logrus.WithFields(logrus.Fields{
		"epoch": uint64(block.Epoch),
		"round": uint64(block.Round),
	}).Debug("Latest committed block found")

	if epoch > uint64(block.Epoch) {
		return false, nil
	}

	if epoch < uint64(block.Epoch) {
		return true, nil
	}

	return round <= uint64(block.Round), nil
}
