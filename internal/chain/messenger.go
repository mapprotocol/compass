package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

type Messenger struct {
	*CommonSync
}

var ignoredSwapTokens = map[common.Address]struct{}{
	common.HexToAddress("0x7046933234A82AF77F14625e8d0fA9Bcc5044a7E"): {},
	common.HexToAddress("0x13CB04d4a5Dfb6398Fc5AB005a6c84337256eE23"): {},
}

func NewMessenger(cs *CommonSync) *Messenger {
	return &Messenger{
		CommonSync: cs,
	}
}

func (m *Messenger) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		if !m.Cfg.SyncToMap && m.Cfg.Id != m.Cfg.MapChainID {
			time.Sleep(time.Hour * 2400)
			return
		}
		if m.Cfg.Filter {
			err := m.filter()
			if err != nil {
				m.Log.Error("Polling blocks failed", "err", err)
			}
			return
		}
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Messenger will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
	var currentBlock = m.Cfg.StartBlock
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			rpcStart := time.Now()
			latestBlock, err := m.Conn.LatestBlock()
			m.State.ObserveRPC("LatestBlock", time.Since(rpcStart).Seconds())
			if err != nil {
				m.State.RecordError("rpc_latest_block", err.Error())
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}
			m.State.SetLatestBlock(latestBlock.Int64())

			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.BlockConfirmations) == -1 {
				m.Log.Debug("Block not ready, will retry", "currentBlock", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BalanceRetryInterval)
				continue
			}
			count, err := m.mosHandler(m, currentBlock)
			if m.Cfg.SkipError && errors.Is(err, NotVerifyAble) {
				m.Log.Info("Block not verify, will ignore", "startBlock", m.Cfg.StartBlock)
				m.Cfg.StartBlock = m.Cfg.StartBlock.Add(m.Cfg.StartBlock, big.NewInt(1))
				err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if err != nil {
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				m.State.RecordError("mos_handler", err.Error())
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if count > 0 {
				m.State.IncEventsMatched(count)
			}

			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

			m.State.SetCurrentBlock(currentBlock.Int64())
			m.State.IncBlocksProcessed(1)
			currentBlock.Add(currentBlock, big.NewInt(1))
			if latestBlock.Int64()-currentBlock.Int64() <= m.Cfg.BlockConfirmations.Int64() {
				time.Sleep(constant.MessengerInterval)
			}
			if currentBlock.Int64()%100 == 0 {
				m.Log.Info("Msger report progress", "latestBlock", latestBlock, "block", currentBlock)
			}
		}
	}
}

func (m *Messenger) filter() error {
	return (&FilterRunner{
		Sync:      m.CommonSync,
		Client:    m.FilterClient(),
		Processor: m,
		Options:   DefaultFilterRunnerOptions(),
	}).Run()
}

func (m *Messenger) HandleFilterBlock(latestBlock uint64) (int, uint64, error) {
	return m.filterMosHandler(latestBlock)
}

func defaultMosHandler(m *Messenger, blockNumber *big.Int) (int, error) {
	count := 0
	for idx, addr := range m.Cfg.McsContract {
		query := m.BuildQuery(addr, m.Cfg.Events, blockNumber, blockNumber)
		logs, err := m.Conn.Client().FilterLogs(context.Background(), query)
		if err != nil {
			return 0, fmt.Errorf("unable to Filter Logs: %w", err)
		}

		m.Log.Debug("event", "blockNumber ", blockNumber, " logs ", len(logs), "mcs", addr, "events", m.Cfg.Events)
		for _, log := range logs {
			ele := log
			send, err := log2Msg(m, &ele, idx)
			if err != nil {
				return 0, err
			}
			count += send
		}
	}
	return count, nil
}

func log2Msg(m *Messenger, log *types.Log, idx int) (int, error) {
	var (
		proofType int64
		toChainID uint64
		err       error
	)

	orderId := log.Topics[1]
	toChainID, _ = strconv.ParseUint(strconv.FormatUint(uint64(m.Cfg.MapChainID), 10), 10, 64)
	if m.Cfg.Id == m.Cfg.MapChainID {
		toChainID = big.NewInt(0).SetBytes(log.Topics[2].Bytes()[8:16]).Uint64()
	}
	chainName, ok := mapprotocol.OnlineChaId[msg.ChainId(toChainID)]
	if !ok {
		m.Log.Info("Map Found a log that is not the current task ", "blockNumber", log.BlockNumber, "toChainID", toChainID)
		return 0, nil
	}
	m.Log.Info("Event found", "blockNumber", log.BlockNumber, "txHash", log.TxHash, "logIdx", log.Index, "toChainID", toChainID, "orderId", orderId)
	if strings.ToLower(chainName) == "tron" || strings.ToLower(chainName) == "sol" {
		proofType = constant.ProofTypeOfLogOracle
	} else {
		proofType, err = PreSendTx(idx, uint64(m.Cfg.Id), toChainID, big.NewInt(0).SetUint64(log.BlockNumber), orderId.Bytes())
		if errors.Is(err, OrderExist) {
			m.Log.Info("This txHash order exist", "txHash", log.TxHash, "toChainID", toChainID)
			return 0, nil
		}
		if errors.Is(err, NotVerifyAble) {
			m.Log.Info("CurrentBlock not verify", "txHash", log.TxHash, "toChainID", toChainID)
			return 0, err
		}
		if err != nil {
			return 0, err
		}
	}

	prepared, err := NewMessageGate(m.CommonSync).Prepare(log, MessageGateOptions{
		Idx:         idx,
		ToChainID:   toChainID,
		OrderID:     orderId,
		ProofType:   proofType,
		MapChainLog: m.Cfg.Id == m.Cfg.MapChainID,
		DoPreSend:   false,
		RequireSign: true,
		LogPrefix:   "Msger",
	})
	if errors.Is(err, OrderIgnored) || errors.Is(err, OrderExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	message, err := m.assembleProof(m, log, proofType, toChainID, prepared.Sign)
	if err != nil {
		return 0, err
	}
	message.Idx = idx
	err = m.Router.Send(*message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return 0, err
	}
	return 1, nil
}

func hasIgnoredSwapToken(log *types.Log, mapChain bool, receiptLogs ...*types.Log) (bool, error) {
	token, _, err := MatchSpecialSwapToken(log, mapChain, receiptLogs...)
	return token != (common.Address{}), err
}

func (m *Messenger) specialTokenDelayReady(log *types.Log) (bool, time.Duration, error) {
	return SpecialTokenDelayReady(m.Conn.Client(), log)
}

func SpecialTokenDelayReady(cli *ethclient.Client, log *types.Log) (bool, time.Duration, error) {
	header, err := cli.HeaderByNumber(context.Background(), big.NewInt(0).SetUint64(log.BlockNumber))
	if err != nil {
		return false, 0, err
	}
	readyAt := time.Unix(int64(header.Time), 0).Add(time.Hour)
	wait := time.Until(readyAt)
	if wait > 0 {
		return false, wait, nil
	}
	return true, 0, nil
}

func ignoredSwapToken(log *types.Log, mapChain bool, receiptLogs ...*types.Log) (common.Address, string, error) {
	return MatchSpecialSwapToken(log, mapChain, receiptLogs...)
}

func MatchSpecialSwapToken(log *types.Log, mapChain bool, receiptLogs ...*types.Log) (common.Address, string, error) {
	var (
		tokens *MessageOutTokens
		err    error
	)
	if mapChain {
		tokens, err = DecodeMessageRelayTokens(log)
	} else {
		tokens, err = DecodeMessageOutTokens(log)
	}
	if err != nil {
		return common.Address{}, "", err
	}
	if _, ok := ignoredSwapTokens[tokens.Token]; ok {
		return tokens.Token, "event_token", nil
	}
	if len(tokens.DstToken) == common.AddressLength {
		dstToken := common.BytesToAddress(tokens.DstToken)
		if _, ok := ignoredSwapTokens[dstToken]; ok {
			return dstToken, "dst_token", nil
		}
	}
	for _, receiptLog := range receiptLogs {
		if _, ok := ignoredSwapTokens[receiptLog.Address]; ok {
			return receiptLog.Address, "receipt_log_address", nil
		}
	}
	return common.Address{}, "", nil
}

func Signer(cli *ethclient.Client, selfId, toId uint64, log *types.Log, proofType int64) (*ProposalInfoResp, error) {
	bn := big.NewInt(int64(log.BlockNumber))
	ret, err := MulSignInfo(0, toId)
	if err != nil {
		return nil, fmt.Errorf("MulSignInfo failed: %w", err)
	}
	header, err := cli.HeaderByNumber(context.Background(), big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		return nil, err
	}
	switch proofType {
	case constant.ProofTypeOfNewOracle:
		genRece, err := genMptReceipt(cli, int64(selfId), bn) //  hash修改
		if err != nil {
			return nil, err
		}
		if genRece != nil {
			header.ReceiptHash = *genRece
		}
	case constant.ProofTypeOfLogOracle:
		hash, _ := GenLogReceipt(log)
		if hash != nil {
			header.ReceiptHash = *hash
		}

		idx := log.Index
		bn = proof.GenLogBlockNumber(bn, log.TxIndex, idx)
	default:
		return nil, fmt.Errorf("unknown proof type %d", proofType)
	}

	piRet, err := ProposalInfo(0, selfId, toId, bn, header.ReceiptHash, ret.Version)
	if err != nil {
		return nil, fmt.Errorf("ProposalInfo failed: %w", err)
	}
	if !piRet.CanVerify {
		return nil, NotVerifyAble
	}
	return piRet, nil
}

func PersonalSign(message string, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return personalSign(message, privateKey)
}

func personalSign(message string, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	fullMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(fullMessage))
	signatureBytes, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, err
	}
	signatureBytes[64] += 27
	return signatureBytes, nil
}

func GetSigner(blockNumber *big.Int, receiptHash common.Hash, selfId, toChainID uint64) (*ProposalInfoResp, error) {
	ret, err := MulSignInfo(0, toChainID)
	if err != nil {
		return nil, err
	}

	piRet, err := ProposalInfo(0, selfId, toChainID, blockNumber, receiptHash, ret.Version)
	if err != nil {
		return nil, err
	}

	if !piRet.CanVerify {
		return nil, NotVerifyAble
	}

	return piRet, nil
}
