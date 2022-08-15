package near

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

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

	return nil
}

// sync function of Messenger will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to RetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
	var currentBlock = m.cfg.startBlock
	m.log.Info("Polling Blocks...", "block", currentBlock)

	var retry = RetryLimit
	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			// No more retries, goto next block
			if retry == 0 {
				m.log.Error("Polling failed, retries exceeded")
				m.sysErr <- ErrFatalPolling
				return nil
			}

			// messager
			// Parse out events
			count, err := m.getEventsForBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to get events for block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(RetryInterval)
				continue
			}

			// hold until all messages are handled
			_ = m.waitUntilMsgHandled(count)

			// Write to block store. Not a critical operation, no need to retry
			err = m.blockStore.StoreBlock(currentBlock)
			if err != nil {
				m.log.Error("Failed to write latest block to blockstore", "block", currentBlock, "err", err)
			}
			if m.metrics != nil {
				m.metrics.BlocksProcessed.Inc()
			}

			m.latestBlock.LastUpdated = time.Now()

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = RetryLimit
			time.Sleep(RetryInterval)
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
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

	//m.log.Info("收到的数据", "result", result)
	//time.Sleep(time.Second * 3)
	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return 0, errors.Wrap(err, "json marshal failed")
	}
	target := make([]mapprotocol.IndexerExecutionOutcomeWithReceipt, 0)
	for _, shard := range data.Shards {
		for _, outcome := range shard.ReceiptExecutionOutcomes {
			if outcome.ExecutionOutcome.Outcome.ExecutorID != m.cfg.bridgeContract {
				continue
			}
			if len(outcome.ExecutionOutcome.Outcome.Logs) == 0 {
				continue
			}
			for _, ls := range outcome.ExecutionOutcome.Outcome.Logs {
				if !strings.HasPrefix(ls, mapprotocol.HashOfTransferOut) && !strings.HasPrefix(ls, mapprotocol.HashOfDepositOut) {
					continue
				}
			}
			m.log.Info("Event found")
			target = append(target, outcome)
		}
	}

	if len(target) == 0 {
		return 0, nil
	}

	ret, err := m.makeMessage(target)
	if err != nil {
		//m.log.Info("make message failed", "err", err)
		cmd := redis.GetClient().RPush(context.Background(), redis.ListKey, result)
		_, err = cmd.Result()
		if err != nil {
			m.log.Error("make message failed, retry insert failed", "err", err)
		}
	}

	return ret, nil
}

func (m *Messenger) makeMessage(target []mapprotocol.IndexerExecutionOutcomeWithReceipt) (int, error) {
	ret := 0
	for _, tg := range target {
		m.log.Info("makeMessage 收到一条数据", "tg", tg)
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
			break
		}

		blkBytes := near.Borshify(blk)
		proofBytes, err := near.BorshifyOutcomeProof(proof)
		if err != nil {
			return 0, errors.Wrap(err, "borshifyOutcomeProof failed")
		}

		all, err := mapprotocol.NearGetBytes.Methods["getBytes"].Inputs.Pack(blkBytes, proofBytes)
		if err != nil {
			return 0, errors.Wrap(err, "getBytes pack failed")
		}

		m.log.Info("near2map的参数", "all", "0x"+common.Bytes2Hex(all))
		//input, err := mapprotocol.Eth2MapTransferInAbi.Pack(mapprotocol.MethodOfTransferIn, new(big.Int).SetUint64(uint64(m.cfg.id)), all)
		//if err != nil {
		//	return 0, errors.Wrap(err, "transferIn pack failed")
		//}
		input, err := mapprotocol.NearVerify.Pack(mapprotocol.MethodVerifyProofData, all)
		if err != nil {
			return 0, errors.Wrap(err, "verifyProof pack failed")
		}
		m.log.Info("near2map的参数，transferIn打包", "input", "0x"+common.Bytes2Hex(input))
		// get fromChainId and toChainId
		logs := strings.SplitN(tg.ExecutionOutcome.Outcome.Logs[0], ":", 2)
		out := near.TransferOut{}
		err = json.Unmarshal([]byte(logs[1]), &out)
		if err != nil {
			return 0, errors.Wrap(err, "logs format failed")
		}

		msgpayload := []interface{}{input}
		message := msg.NewSwapWithProof(msg.ChainId(out.FromChain.Uint64()), m.cfg.mapChainID, msgpayload, m.msgCh)
		err = m.router.Send(message)
		ret++
	}
	return ret, nil
}
