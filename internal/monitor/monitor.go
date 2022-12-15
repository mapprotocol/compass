package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
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

func New(cs *chain.CommonSync) *Monitor {
	return &Monitor{
		CommonSync: cs,
		balance:    new(big.Int),
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

			m.Log.Info("Get balance result", "account", addr, "balance", balance)

			if balance.Cmp(m.balance) != 0 {
				m.balance = balance
				m.timestamp = time.Now().Unix()
			}

			if m.Cfg.Id == m.Cfg.MapChainID {
				if balance.Cmp(constant.MapWaterline) == -1 {
					// alarm
					m.alarm(context.Background(),
						fmt.Sprintf("Balance Less than five yuan,\nchain=%s,addr=%s,balance=%d", m.Cfg.Name, m.Cfg.From,
							balance.Div(balance, constant.Wei)))
				}
			} else {
				if balance.Cmp(constant.Waterline) == -1 {
					// alarm
					m.alarm(context.Background(),
						fmt.Sprintf("Balance Less than five yuan,\nchain=%s,addr=%s,balance=%d", m.Cfg.Name, m.Cfg.From,
							balance.Div(balance, constant.Wei)))
				}
			}

			if (time.Now().Unix() - m.timestamp) > constant.AlarmMinute {
				// alarm
				m.alarm(context.Background(),
					fmt.Sprintf("No transaction occurred in addr in the last %d seconds,\n"+
						"chain=%s,addr=%s,balance=%d", constant.AlarmMinute, m.Cfg.Name, m.Cfg.From,
						balance.Div(balance, constant.Wei)))
			}

			time.Sleep(constant.BalanceRetryInterval)
		}
	}
}

func (m *Monitor) alarm(ctx context.Context, msg string) {
	body, err := json.Marshal(map[string]interface{}{
		"text": msg,
	})
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "POST", m.Cfg.HooksUrl, ioutil.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		m.Log.Warn("read resp failed", "err", err)
		return
	}
	m.Log.Info("send alarm message", "resp", string(data))
}
