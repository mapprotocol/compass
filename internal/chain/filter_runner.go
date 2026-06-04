package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

type FilterHandler func(latestBlock uint64) (int, uint64, error)

type FilterProcessor interface {
	HandleFilterBlock(latestBlock uint64) (int, uint64, error)
}

type FilterProcessorFunc func(latestBlock uint64) (int, uint64, error)

func (f FilterProcessorFunc) HandleFilterBlock(latestBlock uint64) (int, uint64, error) {
	return f(latestBlock)
}

type FilterRunnerOptions struct {
	TerminateError     error
	NotVerifyableSleep time.Duration
	SkipMissingField   bool
}

type FilterRunner struct {
	Sync      *CommonSync
	Client    FilterClient
	Processor FilterProcessor
	Options   FilterRunnerOptions
}

func DefaultFilterRunnerOptions() FilterRunnerOptions {
	return FilterRunnerOptions{
		TerminateError:     errors.New("filter polling terminated"),
		NotVerifyableSleep: constant.BlockRetryInterval,
		SkipMissingField:   true,
	}
}

func RunFilterLoop(cs *CommonSync, handler FilterHandler, opts FilterRunnerOptions) error {
	runner := &FilterRunner{
		Sync:      cs,
		Client:    cs.FilterClient(),
		Processor: FilterProcessorFunc(handler),
		Options:   opts,
	}
	return runner.Run()
}

func (r *FilterRunner) Run() error {
	cs := r.Sync
	opts := r.Options
	if opts.TerminateError == nil {
		opts.TerminateError = errors.New("filter polling terminated")
	}
	if opts.NotVerifyableSleep == 0 {
		opts.NotVerifyableSleep = constant.BlockRetryInterval
	}
	if r.Client == nil {
		return fmt.Errorf("filter client is nil")
	}

	for {
		select {
		case <-cs.Stop:
			return opts.TerminateError
		default:
			rpcStart := time.Now()
			latestBlock, err := r.latestBlock()
			cs.State.ObserveRPC("FilterLatestBlock", time.Since(rpcStart).Seconds())
			if err != nil {
				cs.State.RecordError("rpc_filter_latest", err.Error())
				cs.Log.Error("Unable to get latest block", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			cs.State.SetLatestBlock(latestBlock.Int64())
			count, progressBlock, err := r.Processor.HandleFilterBlock(latestBlock.Uint64())
			if cs.Cfg.SkipError && errors.Is(err, NotVerifyAble) {
				cs.Log.Info("Block not verify, will ignore", "startBlock", cs.Cfg.StartBlock)
				cs.Cfg.StartBlock = cs.Cfg.StartBlock.Add(cs.Cfg.StartBlock, big.NewInt(1))
				_ = cs.BlockStore.StoreBlock(cs.Cfg.StartBlock)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if err != nil {
				cs.Log.Error("Filter Failed to get events for block", "err", err)
				if errors.Is(err, NotVerifyAble) {
					time.Sleep(opts.NotVerifyableSleep)
					continue
				}
				if opts.SkipMissingField && strings.Contains(err.Error(), "missing required field") {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				util.Alarm(context.Background(), fmt.Sprintf("filter mos failed, chain=%s, err is %s", cs.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if progressBlock > 0 {
				cs.State.SetCurrentBlock(int64(progressBlock))
			}

			_ = cs.WaitUntilMsgHandled(count)
			if err := cs.BlockStore.StoreBlock(cs.Cfg.StartBlock); err != nil {
				cs.Log.Error("Filter Failed to write latest block to blockStore", "err", err)
			}
			if count > 0 {
				cs.State.IncEventsMatched(count)
			}
			cs.State.IncBlocksProcessed(1)

			time.Sleep(constant.MessengerInterval)
		}
	}
}

func (r *FilterRunner) latestBlock() (*big.Int, error) {
	cs := r.Sync
	if time.Now().Unix()-cs.reqTime < constant.ReqInterval {
		return big.NewInt(cs.cacheBlockNumber), nil
	}
	latestBlock, err := r.Client.LatestBlock(int64(cs.Cfg.Id))
	if err != nil {
		time.Sleep(constant.BlockRetryInterval)
		return nil, err
	}
	cs.Log.Debug("Filter latest block", "block", latestBlock)
	cs.cacheBlockNumber = latestBlock.Int64()
	cs.reqTime = time.Now().Unix()
	return latestBlock, nil
}
