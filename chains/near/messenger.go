package near

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

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

			latestBlock, err := m.conn.LatestBlock()
			if err != nil {
				m.log.Error("Unable to get latest block", "block", currentBlock, "err", err)
				retry--
				time.Sleep(BlockRetryInterval)
				continue
			}

			if m.metrics != nil {
				m.metrics.LatestKnownBlock.Set(float64(latestBlock.Int64()))
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
				m.metrics.LatestProcessedBlock.Set(float64(latestBlock.Int64()))
			}

			m.latestBlock.Height = big.NewInt(0).Set(latestBlock)
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
	// querying for logs
	ctx := context.Background()
	cmd := redis.GetClient().RPop(ctx, redis.ListKey)
	result, err := cmd.Result()
	if err != nil && !errors.Is(err, rds.Nil) {
		return 0, errors.Wrap(err, "lPop failed")
	}

	if err != nil && errors.Is(err, rds.Nil) {
		return 0, nil
	}
	fmt.Printf("收到的数据， %v \n", result)

	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal([]byte(result), &data)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshal failed")
	}

	m.log.Info("获取的消息", "msg", data)

	// step2:组装 struct
	//proof := mapprotocol.NearReceiptProof{
	//	BlockHeaderLite:  mapprotocol.BlockHeaderLite{},
	//	BlockProof:       nil,
	//	OutcomeProof:     mapprotocol.OutcomeProof{},
	//	OutcomeRootProof: nil,
	//}

	return 0, nil
}
