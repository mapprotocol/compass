package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

type Monitor struct {
	*chain.CommonSync
	balance   *big.Int
	timestamp int64
}

func NewMonitor(cs *chain.CommonSync) *Monitor {
	return &Monitor{
		CommonSync: cs,
	}
}

func (m *Monitor) Sync() error {
	m.Log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.Log.Error("Polling Account balance failed", "err", err)
		}
	}()

	return nil
}

// sync function of Monitor will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Monitor) sync() error {
	addr := common.HexToAddress(m.Cfg.From)
	for {
		select {
		case <-m.Stop:
			return errors.New("polling terminated")
		default:
			balance, err := m.Conn.Client().BalanceAt(context.Background(), addr, nil)
			if err != nil {
				m.Log.Error("Unable to get user balance failed", "from", addr, "err", err)
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			m.Log.Info("get balance result", "account", addr, "balance", balance)

			if balance.Cmp(m.balance) != 0 {
				m.balance = balance
				m.timestamp = time.Now().Unix()
			}

			if balance.Cmp(constant.Waterline) == -1 {
				// alarm
			}

			if (time.Now().Unix() - m.timestamp) > constant.TenMinute {
				// alarm
			}

			time.Sleep(constant.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) doRequest(ctx context.Context, msg interface{}) (io.ReadCloser, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, ioutil.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(body))

	// set headers
	c.mu.Lock()
	req.Header = c.headers.Clone()
	c.mu.Unlock()

	// do request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		//var buf bytes.Buffer
		//var body []byte
		//if _, err := buf.ReadFrom(resp.Body); err == nil {
		//	body = buf.Bytes()
		//}

		return nil, nil
		//return nil, HTTPError{
		//	Status:     resp.Status,
		//	StatusCode: resp.StatusCode,
		//	Body:       body,
		//}
	}
	return resp.Body, nil
}
