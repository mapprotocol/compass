package near

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"

	rds "github.com/go-redis/redis/v8"
	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/redis"
	"github.com/mapprotocol/near-api-go/pkg/client"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"github.com/pkg/errors"
)

type Messenger struct {
	*CommonListen
}

func NewMessenger(cs *CommonListen) *Messenger {
	return &Messenger{
		CommonListen: cs,
	}
}

func (m *Messenger) Sync() error {
	m.log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.log.Error("Polling blocks failed", "err", err)
		}
	}()
	go func() {
		m.watchDog()
	}()

	return nil
}

func (m *Messenger) sync() error {
	var currentBlock = m.cfg.StartBlock

	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.QueryRetryInterval)
				continue
			}

			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(m.blockConfirmations) == -1 {
				m.log.Debug("Block not ready, will retry", "target", currentBlock, "latest", latestBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			count, err := m.getEventsForBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				time.Sleep(RetryInterval)
				util.Alarm(context.Background(), fmt.Sprintf("near mos failed, err is %s", err.Error()))
				continue
			}

			// hold until all messages are handled
			_ = m.waitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = m.blockStore.StoreBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}
			m.latestBlock.LastUpdated = time.Now()

			currentBlock.Add(currentBlock, big.NewInt(1))
		}
	}
}

func (m *Messenger) watchDog() {
	record := ""
	for {
		time.Sleep(time.Minute * 3)
		ctx := context.Background()
		cmd := redis.GetClient().Get(ctx, redis.BlockHeight)
		result, err := cmd.Result()
		if err != nil && !errors.Is(err, rds.Nil) {
			continue
		}
		m.log.Info("Near watchdog scan report", "current", result, "record", record)
		if record != result {
			record = result
			continue
		}
		if record == result {
			util.Alarm(context.Background(), fmt.Sprintf("near scan no change in one minute, please admin handler, now=%s", result))
		}
	}
}

func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
	if !m.cfg.SyncToMap {
		return 0, nil
	}
	// querying for logs
	ctx := context.Background()
	cmd := redis.GetClient().RPop(ctx, redis.ListKey)
	result, err := cmd.Result()
	if err != nil && !errors.Is(err, rds.Nil) {
		return 0, errors.Wrap(err, "rPop failed")
	}

	if err != nil && errors.Is(err, rds.Nil) {
		return 0, nil
	}

	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return 0, errors.Wrap(err, "json marshal failed")
	}
	target := make([]mapprotocol.IndexerExecutionOutcomeWithReceipt, 0)
	for _, shard := range data.Shards {
		for _, outcome := range shard.ReceiptExecutionOutcomes {
			if m.Idx(outcome.ExecutionOutcome.Outcome.ExecutorID) == -1 {
				continue
			}
			if len(outcome.ExecutionOutcome.Outcome.Logs) == 0 {
				continue
			}
			match := false
			for _, ls := range outcome.ExecutionOutcome.Outcome.Logs {
				if !match {
					match = m.match(ls)
				}
			}
			if match {
				m.log.Info("Event found", "log", outcome.ExecutionOutcome.Outcome.Logs, "contract", outcome.ExecutionOutcome.Outcome.ExecutorID)
				target = append(target, outcome)
			} else {
				m.log.Info("Event Not Match", "log", outcome.ExecutionOutcome.Outcome.Logs)
			}
		}
	}

	if len(target) == 0 {
		return 0, nil
	}

	ret, err := m.makeMessage(target)
	if err != nil {
		m.log.Error("make message failed", "err", err)
		cmd := redis.GetClient().RPush(context.Background(), redis.ListKey, result)
		_, err = cmd.Result()
		if err != nil {
			m.log.Error("make message failed, retry insert failed", "err", err)
		}
		time.Sleep(constant.TxRetryInterval)
	}

	return ret, nil
}

func (m *Messenger) match(log string) bool {
	for _, e := range m.cfg.events {
		if strings.HasPrefix(log, e) {
			return true
		}
	}

	return false
}

func (m *Messenger) Idx(contract string) int {
	ret := -1
	for idx, addr := range m.cfg.mcsContract {
		if addr == contract {
			ret = idx
			break
		}
	}

	return ret
}

func (m *Messenger) makeMessage(target []mapprotocol.IndexerExecutionOutcomeWithReceipt) (int, error) {
	ret := 0
	for _, tg := range target {
		m.log.Debug("makeMessage receive one message", "tg", tg)
		time.Sleep(time.Second * 3)
		var (
			err        error
			retryCount = 0
			blk        client.LightClientBlockView
			proof      client.RpcLightClientExecutionProofResponse
		)
		for {
			retryCount++
			if retryCount == RetryLimit {
				return 0, errors.New("make message, retries exceeded")
			}
			blk, err = m.conn.Client().NextLightClientBlock(context.Background(), tg.ExecutionOutcome.BlockHash)
			if err != nil {
				m.log.Warn("get nextLightClientBlock failed, will retry", "err", err)
				time.Sleep(RetryInterval)
				continue
			}

			clientHead, err := m.conn.Client().BlockDetails(context.Background(), block.BlockID(blk.InnerLite.Height))
			if err != nil {
				m.log.Warn("get blockDetails failed, will retry", "err", err)
				time.Sleep(RetryInterval)
				continue
			}

			proof, err = m.conn.Client().LightClientProof(context.Background(), nearclient.Receipt{
				ReceiptID:       tg.ExecutionOutcome.ID,
				ReceiverID:      tg.Receipt.ReceiverID,
				LightClientHead: clientHead.Header.Hash,
			})
			if err != nil {
				m.log.Warn("get lightClientProof failed, will retry", "err", err)
				time.Sleep(RetryInterval)
				continue
			}
			if len(proof.BlockProof) <= 0 {
				time.Sleep(RetryInterval)
				continue
			}
			break
		}

		blkBytes := near.Borshify(blk)
		proofBytes, err := near.BorshifyOutcomeProof(proof)
		if err != nil {
			return 0, errors.Wrap(err, "borshifyOutcomeProof failed")
		}

		all, err := mapprotocol.Near.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(blkBytes, proofBytes)
		if err != nil {
			return 0, errors.Wrap(err, "getBytes pack failed")
		}

		// get fromChainId and toChainId
		logs := strings.SplitN(tg.ExecutionOutcome.Outcome.Logs[0], ":", 2)
		out := near.TransferOut{}
		err = json.Unmarshal([]byte(logs[1]), &out)
		if err != nil {
			return 0, errors.Wrap(err, "logs format failed")
		}

		method := mapprotocol.MethodOfTransferIn
		if strings.HasPrefix(tg.ExecutionOutcome.Outcome.Logs[1], mapprotocol.NearOfDepositIn) {
			method = mapprotocol.MethodOfDepositIn
		} else if strings.HasPrefix(tg.ExecutionOutcome.Outcome.Logs[1], mapprotocol.NearOfSwapIn) {
			method = mapprotocol.MethodOfSwapIn
		}
		input, err := mapprotocol.Mcs.Pack(method, new(big.Int).SetUint64(uint64(m.cfg.Id)), all)
		if err != nil {
			return 0, errors.Wrap(err, "transferIn pack failed")
		}

		ids := common.HexToHash(out.OrderId)
		orderId := make([]byte, 0, len(ids))
		for _, id := range ids {
			orderId = append(orderId, id)
		}
		msgPayload := []interface{}{input, orderId, 0, tg.ExecutionOutcome.Outcome.ReceiptIDs}
		message := msg.NewSwapWithProof(m.cfg.Id, m.cfg.MapChainID, msgPayload, m.msgCh)
		message.Idx = m.Idx(tg.ExecutionOutcome.Outcome.ExecutorID)
		err = m.router.Send(message)
		ret++
	}
	return ret, nil
}
