package near

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/msg"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/near"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/mapprotocol/compass/mapprotocol"

	rds "github.com/go-redis/redis/v8"
	"github.com/mapprotocol/compass/pkg/redis"
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
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Messenger) sync() error {
	var currentBlock = m.cfg.startBlock
	m.log.Info("Polling Blocks...", "block", currentBlock)

	var retry = BlockRetryLimit
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
				time.Sleep(BlockRetryInterval)
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
			retry = BlockRetryLimit
			time.Sleep(BlockRetryInterval)
		}
	}
}

// getEventsForBlock looks for the deposit event in the latest block
func (m *Messenger) getEventsForBlock(latestBlock *big.Int) (int, error) {
	log.Println("--------------- ", m.cfg.bridgeContract)
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
	fmt.Printf("收到的数据， %v \n", result)

	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	target := make([]mapprotocol.IndexerExecutionOutcomeWithReceipt, 0)
	for _, shard := range data.Shards {
		for _, outcome := range shard.ReceiptExecutionOutcomes {
			if outcome.ExecutionOutcome.Outcome.ExecutorID != m.cfg.bridgeContract { //  "mcs.pandarr.testnet" { // 合约地址
				continue
			}
			if len(outcome.ExecutionOutcome.Outcome.Logs) == 0 {
				continue
			}
			for _, ls := range outcome.ExecutionOutcome.Outcome.Logs {
				splits := strings.Split(ls, ":")
				if len(splits) != 2 {
					continue
				}
				if !existInSlice(splits[0], mapprotocol.NearEventType) {
					continue
				}
				m.log.Info("find one log", "log", ls)
				//if !strings.HasPrefix(ls, mapprotocol.HashOfTransferOut) && !strings.HasPrefix(ls, mapprotocol.HashOfDepositOut) {
				//	continue
				//}
			}

			target = append(target, outcome)
		}
		//fmt.Println()
	}

	if len(target) == 0 {
		return 0, nil
	}

	m.log.Info("获取的消息", "msg", data)
	for _, tg := range target {
		blk, err := m.conn.Client().NextLightClientBlock(context.Background(), tg.ExecutionOutcome.BlockHash)
		if err != nil {
			return 0, errors.Wrap(err, "nextLightClientBlock failed")
		}

		clientHead, err := m.conn.Client().BlockDetails(context.Background(), block.BlockID(blk.InnerLite.Height))
		if err != nil {
			m.log.Error("BlockDetails failed, err %v", err)
		}

		proof, err := m.conn.Client().LightClientProof(context.Background(), nearclient.Receipt{
			ReceiptID:       tg.ExecutionOutcome.ID,
			ReceiverID:      tg.Receipt.ReceiverID,
			LightClientHead: clientHead.Header.Hash,
		})
		if err != nil {
			m.log.Error("LightClientProof failed, err %v", err)
		}

		blkBytes := near.Borshify(blk)
		m.log.Info("blockBytes, 0x%v", common.Bytes2Hex(blkBytes))
		proofBytes, err := near.BorshifyOutcomeProof(proof)
		if err != nil {
			return 0, errors.Wrap(err, "borshifyOutcomeProof failed")
		}
		m.log.Info("proofBytes", "0x"+common.Bytes2Hex(proofBytes))

		all, err := mapprotocol.NearGetBytes.Methods["getBytes"].Inputs.Pack(blkBytes, proofBytes)
		if err != nil {
			return 0, errors.Wrap(err, "getBytes failed")
		}
		fmt.Println("请求参数 ---------- ", "0x"+common.Bytes2Hex(all))
		input, err := mapprotocol.NearVerify.Methods[mapprotocol.MethodVerifyProofData].Inputs.Pack(all)
		if err != nil {
			return 0, errors.Wrap(err, "verifyProof failed")
		}

		msgpayload := []interface{}{input}
		message := msg.NewSwapWithProof(msg.ChainId(1313161555), msg.ChainId(212), msgpayload, m.msgCh)
		m.log.Info("Event found")
		err = m.router.Send(message)
	}
	return len(target), nil
}

func existInSlice(target string, dst []string) bool {
	for _, d := range dst {
		if target == d {
			return true
		}
	}

	return false
}
