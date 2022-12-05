package near

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
)

type Monitor struct {
	*CommonListen
	balance   *big.Int
	timestamp int64
}

func NewMonitor(cs *CommonListen) *Monitor {
	return &Monitor{
		CommonListen: cs,
		balance:      new(big.Int),
	}
}

func (m *Monitor) Sync() error {
	m.log.Debug("Starting listener...")
	go func() {
		err := m.sync()
		if err != nil {
			m.log.Error("Polling blocks failed", "err", err)
		}
	}()

	return nil
}

// sync function of Monitor will poll for the latest block and listen the log information of transactions in the block
// Polling begins at the block defined in `m.Cfg.startBlock`. Failed attempts to fetch the latest block or parse
// a block will be retried up to BlockRetryLimit times before continuing to the next block.
// However，an error in synchronizing the log will cause the entire program to block
func (m *Monitor) sync() error {
	for {
		select {
		case <-m.stop:
			return errors.New("polling terminated")
		default:
			resp, err := m.conn.Client().AccountView(context.Background(), m.cfg.from, block.FinalityFinal())
			if err != nil {
				m.log.Error("Unable to get user balance failed", "from", m.cfg.from, "err", err)
				time.Sleep(constant.RetryLongInterval)
				continue
			}

			m.log.Info("Get balance result", "account", m.cfg.from, "balance", resp.Amount.String())

			v, ok := new(big.Int).SetString(resp.Amount.String(), 10)
			if ok && v.Cmp(m.balance) != 0 {
				m.balance = v
				m.timestamp = time.Now().Unix()
			}

			v = v.Div(v, constant.WeiOfNear)
			if v.Cmp(constant.WaterlineOfNear) == -1 {
				// alarm
				m.alarm(context.Background(),
					fmt.Sprintf("Balance Less than five yuan,\nchain=%s,addr=%s,balance=%d", m.cfg.name, m.cfg.from,
						v.Div(v, constant.WeiOfNear)))
			}

			if (time.Now().Unix() - m.timestamp) > constant.AlarmMinute {
				// alarm
				m.alarm(context.Background(),
					fmt.Sprintf("No transaction occurred in addr in the last %d seconds,\n"+
						"chain=%s,addr=%s,balance=%d", constant.AlarmMinute, m.cfg.name, m.cfg.from,
						v.Div(v, constant.Wei)))
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
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://hooks.slack.com/services/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", ioutil.NopCloser(bytes.NewReader(body)))
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
		m.log.Warn("read resp failed", "err", err)
		return
	}
	m.log.Info("send alarm message", "resp", string(data))
}
