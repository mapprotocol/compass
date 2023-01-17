package near

import (
	"context"
	"errors"
	"fmt"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"math/big"
	"time"
)

type Monitor struct {
	*CommonListen
	balance, syncedHeight      *big.Int
	timestamp, heightTimestamp int64
}

func NewMonitor(cs *CommonListen) *Monitor {
	return &Monitor{
		CommonListen: cs,
		balance:      new(big.Int),
		syncedHeight: new(big.Int),
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
				util.Alarm(context.Background(),
					fmt.Sprintf("Balance Less than %d Near \nchain=%s addr=%s near=%d", waterLine.Int64(),
						m.cfg.name, m.cfg.from, conversion.Int64()))
			}

			if (time.Now().Unix() - m.timestamp) > changeInterval.Int64() {
				time.Sleep(time.Second * 5)
				// alarm
				util.Alarm(context.Background(),
					fmt.Sprintf("No transaction occurred in addr in the last %d seconds,\n"+
						"chain=%s addr=%s near=%d", changeInterval.Int64(), m.cfg.name, m.cfg.from,
						v.Div(v, constant.WeiOfNear)))
			}

			height, err := mapprotocol.Get2MapHeight(m.cfg.id)
			m.log.Info("Check Height", "syncHeight", height, "record", m.syncedHeight)
			if err != nil {
				m.log.Error("get2MapHeight failed", "err", err)
			} else {
				if height.Cmp(m.syncedHeight) != 0 {
					m.syncedHeight = height
					m.heightTimestamp = time.Now().Unix()
				}
				if (time.Now().Unix() - m.heightTimestamp) > changeInterval.Int64() {
					time.Sleep(time.Second * 30)
					// alarm
					util.Alarm(context.Background(),
						fmt.Sprintf("Near2Map height in %d seconds no change, height=%d\n", m.syncedHeight.Uint64()))
				}
			}

			time.Sleep(constant.BalanceRetryInterval)
		}
	}
}
