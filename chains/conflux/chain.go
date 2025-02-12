package conflux

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/pkg/abi"
	"github.com/mapprotocol/compass/pkg/contract"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common/hexutil"
	connection "github.com/mapprotocol/compass/connections/ethereum"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/conflux"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	skippedRound, startNumber uint64
	cli                       = &conflux.Client{}
)

type Chain struct {
}

func New() *Chain {
	return &Chain{}
}

func (c *Chain) New(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, role mapprotocol.Role) (core.Chain, error) {
	client, err := conflux.NewClient(chainCfg.Opts[chain.Eth2Url])
	if err != nil {
		panic("conflux init client failed" + err.Error())
	}
	cli = client
	return chain.New(chainCfg, logger, sysErr, role, connection.NewConnection,
		chain.OptOfSync2Map(c.syncHeaderToMap),
		chain.OptOfInitHeight(mapprotocol.HeaderOneCount),
		chain.OptOfAssembleProof(c.assembleProof),
		chain.OptOfOracleHandler(chain.DefaultOracleHandler),
	)
}

func (c *Chain) syncHeaderToMap(m *chain.Maintainer, latestBlock *big.Int) error {
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
		m.Log.Info("conflux update endBlock", "startNumber", startNumber, "endNumber", state.FinalizedBlockNumber.Uint64())
		if state.FinalizedBlockNumber.Uint64()-startNumber > 200 {
			err = updateHeaders(m, startNumber, state.FinalizedBlockNumber.Uint64())
			if err != nil {
				return err
			}
		}
		startNumber = state.FinalizedBlockNumber.Uint64()
	}

	committed, err := c.isCommitted(epoch, round)
	if err != nil {
		m.Log.Error("Conflux isCommitted Failed", "err", err)
		return err
	}
	m.Log.Info("Conflux isCommitted", "committed", committed, "epoch", epoch, "round", round)
	if !committed {
		time.Sleep(time.Second * 5)
		return nil
	}

	ledger, err := cli.GetLedgerInfoByEpochAndRound(
		context.Background(),
		hexutil.Uint64(epoch),
		hexutil.Uint64(round),
	)
	if err != nil && err.Error() != "not found" {
		return errors.WithMessage(err, "Failed to get ledger")
	}

	// no ledger in round, just skip it
	if ledger == nil {
		m.Log.Info("No ledger info in this round", "epoch", epoch)
		skippedRound = round
		time.Sleep(time.Second * 5)
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

	input, err := mapprotocol.Conflux.Methods["relayPOS"].Inputs.Pack(conflux.ConvertLedger(ledger))
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

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

func (c *Chain) assembleProof(m *chain.Messenger, log *types.Log, proofType int64, toChainID uint64, sign [][]byte) (*msg.Message, error) {
	var (
		err     error
		message msg.Message
		orderId = log.Topics[1]
	)

	var orderId32 [32]byte
	for idx, v := range orderId {
		orderId32[idx] = v
	}

	payload, err := c.Proof(m.Conn.Client(), log, "", proofType, uint64(m.Cfg.Id), toChainID, sign)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	msgPayload := []interface{}{payload, orderId32, log.BlockNumber, log.TxHash}
	message = msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, msgPayload, m.MsgCh)
	return &message, nil
}

func getState(m *chain.Maintainer) (*conflux.ILightNodeState, error) {
	data, err := mapprotocol.GetDataByManager(mapprotocol.MethodClientState, big.NewInt(int64(m.Cfg.Id)))
	if err != nil {
		return nil, err
	}
	analysis, err := mapprotocol.Other.Methods[mapprotocol.MethodOfState].Outputs.Unpack(data)
	if err != nil {
		return nil, errors.Wrap(err, "analysis")
	}
	ret := new(conflux.ILightNodeState)
	if err = mapprotocol.Other.Methods[mapprotocol.MethodOfState].Outputs.Copy(&ret, analysis); err != nil {
		return nil, errors.Wrap(err, "analysis copy")
	}
	return ret, nil
}

func nearestPivot(selfId uint64, height *big.Int) (*big.Int, error) {
	pack, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfNearestPivot].Inputs.Pack(height)
	if err != nil {
		return nil, errors.Wrap(err, "nearestPivot pack failed")
	}
	data, err := mapprotocol.GetDataByManager(mapprotocol.MethodOFinalizedState, big.NewInt(int64(selfId)), pack)
	if err != nil {
		return nil, errors.Wrap(err, "finalizedState unpack failed")
	}
	analysis, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfNearestPivot].Outputs.Unpack(data)
	if err != nil {
		return nil, errors.Wrap(err, "nearestPivot unpack failed")
	}
	ret := new(big.Int)
	if err = mapprotocol.Conflux.Methods[mapprotocol.MethodOfNearestPivot].Outputs.Copy(&ret, analysis); err != nil {
		return nil, errors.Wrap(err, "nearestPivot copy failed")
	}
	return ret, nil
}

func updateHeaders(m *chain.Maintainer, startNumber, endNumber uint64) error {
	m.Log.Info("Sync Header", "startNumber", startNumber, "endNumber", endNumber)
	headers := make([][]byte, mapprotocol.HeaderLengthOfConflux)
	idx := mapprotocol.HeaderLengthOfConflux - 1
	for i := endNumber; i >= startNumber; i-- {
		blk, err := cli.GetBlockByEpochNumber(context.Background(), hexutil.Uint64(i))
		if err != nil {
			return err
		}

		m.Log.Info("updateHeaders ", "height", i, "idx", idx)
		ele := conflux.MustRLPEncodeBlock(blk)
		headers[idx] = ele
		idx--
		if idx != -1 && i != startNumber {
			continue
		}
		if i == startNumber {
			headers = headers[idx+1:]
		}

		input, err := mapprotocol.Conflux.Methods[mapprotocol.MethodOfGetBlockHeadersBytes].Inputs.Pack(headers)
		if err != nil {
			m.Log.Error("Failed to header abi pack", "err", err)
			return err
		}
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
		if i != startNumber {
			i++ // special handle
		}
		idx = mapprotocol.HeaderLengthOfConflux - 1
		time.Sleep(time.Second * 2)
	}

	return nil
}

func (c *Chain) isCommitted(epoch, round uint64) (bool, error) {
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

func (c *Chain) Connect(id, endpoint, mcs, lightNode, oracleNode string) (*ethclient.Client, error) {
	conn := connection.NewConnection(endpoint, true, nil, nil, big.NewInt(chain.DefaultGasLimit),
		big.NewInt(chain.DefaultGasPrice), chain.DefaultGasMultiplier)
	err := conn.Connect()
	if err != nil {
		return nil, err
	}

	fn := sync.OnceFunc(func() {
		idInt, _ := strconv.ParseUint(id, 10, 64)
		oracleAbi, _ := abi.New(mapprotocol.OracleAbiJson)
		call := contract.New(conn, []common.Address{common.HexToAddress(mcs)}, oracleAbi)
		mapprotocol.ContractMapping[msg.ChainId(idInt)] = call

		oAbi, _ := abi.New(mapprotocol.SignerJson)
		oracleCall := contract.New(conn, []common.Address{common.HexToAddress(oracleNode)}, oAbi)
		mapprotocol.SingMapping[msg.ChainId(idInt)] = oracleCall

		fn := mapprotocol.Map2EthHeight(constant.ZeroAddress.Hex(), common.HexToAddress(lightNode), conn.Client())
		mapprotocol.Map2OtherHeight[msg.ChainId(idInt)] = fn
	})
	fn()

	return conn.Client(), nil
}

func (c *Chain) Proof(client *ethclient.Client, log *types.Log, endpoint string, proofType int64, selfId,
	toChainID uint64, sign [][]byte) ([]byte, error) {
	var (
		err     error
		pivot   = big.NewInt(0)
		orderId = log.Topics[1]
		method  = chain.GetMethod(log.Topics[0])
	)
	if proofType == constant.ProofTypeOfOrigin {
		pivot, err = nearestPivot(selfId, new(big.Int).SetUint64(log.BlockNumber+conflux.DeferredExecutionEpochs))
		if err != nil {
			return nil, err
		}
	}
	var orderId32 [32]byte
	for idx, v := range orderId {
		orderId32[idx] = v
	}

	txsHash, err := mapprotocol.GetTxsByBn(client, big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(client, txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	ret, err := conflux.AssembleProof(cli, pivot.Uint64(), uint64(proofType), method, msg.ChainId(selfId), log, receipts, orderId32, sign)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse Log: %w", err)
	}

	return ret, nil
}

func (c *Chain) Maintainer(client *ethclient.Client, selfId, toChainId uint64, srcEndpoint string) ([]byte, error) {
	return nil, errors.New("cfx temporary not support maintainer")
}
