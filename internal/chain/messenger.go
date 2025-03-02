package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/util"
)

type Messenger struct {
	*CommonSync
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
			latestBlock, err := m.Conn.LatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}

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
				continue
			}
			if err != nil {
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				m.Log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}

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
	for {
		select {
		case <-m.Stop:
			return errors.New("filter polling terminated")
		default:
			latestBlock, err := m.FilterLatestBlock()
			if err != nil {
				m.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			count, err := m.filterMosHandler(latestBlock.Uint64())
			if m.Cfg.SkipError && errors.Is(err, NotVerifyAble) {
				m.Log.Info("Block not verify, will ignore", "startBlock", m.Cfg.StartBlock)
				m.Cfg.StartBlock = m.Cfg.StartBlock.Add(m.Cfg.StartBlock, big.NewInt(1))
				err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
				continue
			}
			if err != nil {
				m.Log.Error("Filter Failed to get events for block", "err", err)
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				if strings.Index(err.Error(), "missing required field") != -1 {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				util.Alarm(context.Background(), fmt.Sprintf("filter mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			// hold until all messages are handled
			_ = m.WaitUntilMsgHandled(count)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Filter Failed to write latest block to blockStore", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
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
	if strings.ToLower(chainName) == "near" {
		proofType = 1
	} else if strings.ToLower(chainName) == "tron" || strings.ToLower(chainName) == "sol" {
		proofType = constant.ProofTypeOfLogOracle
	} else if strings.ToLower(chainName) == "ton" {
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

	var sign [][]byte
	if proofType == constant.ProofTypeOfNewOracle || proofType == constant.ProofTypeOfLogOracle {
		ret, err := Signer(m.Conn.Client(), uint64(m.Cfg.Id), uint64(m.Cfg.MapChainID), log, proofType)
		if err != nil {
			return 0, err
		}
		sign = ret.Signatures
	}

	tmpLog := log
	message, err := m.assembleProof(m, tmpLog, proofType, toChainID, sign)
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
		if selfId != constant.CfxChainId && selfId != constant.MapChainId {
			idx = 0
		}
		bn = proof.GenLogBlockNumber(bn, idx)
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
