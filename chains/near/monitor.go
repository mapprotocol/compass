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
	"os"
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
	env := os.Getenv("compass")
	waterLine, ok := new(big.Int).SetString(m.cfg.WaterLine, 10)
	if !ok {
		m.sysErr <- errors.New("near waterLine Not Number")
		return nil
	}
	waterLine = waterLine.Div(waterLine, constant.WeiOfNear)
	changeInterval, ok := new(big.Int).SetString(m.cfg.ChangeInterval, 10)
	if !ok {
		m.sysErr <- errors.New("near changeInterval Not Number")
		return nil
	}
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

			conversion := new(big.Int).Div(v, constant.WeiOfNear)
			if conversion.Cmp(waterLine) == -1 {
				// alarm
				m.alarm(context.Background(),
					fmt.Sprintf("%s Balance Less than %d Near \nchain=%s addr=%s near=%d", env, waterLine.Int64(),
						m.cfg.name, m.cfg.from, conversion.Int64()))
			}

			m.log.Info("Get balance result", "timeCha", time.Now().Unix()-m.timestamp,
				"changeInterval", changeInterval.Int64())
			if (time.Now().Unix() - m.timestamp) > changeInterval.Int64() {
				time.Sleep(time.Second * 30)
				// alarm
				m.alarm(context.Background(),
					fmt.Sprintf("%s No transaction occurred in addr in the last %d seconds,\n"+
						"chain=%s addr=%s near=%d", env, changeInterval.Int64(), m.cfg.name, m.cfg.from,
						v.Div(v, constant.WeiOfNear)))
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
	req, err := http.NewRequestWithContext(ctx, "POST", m.cfg.HooksUrl, ioutil.NopCloser(bytes.NewReader(body)))
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
