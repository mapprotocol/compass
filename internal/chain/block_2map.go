package chain

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

// execToMapMsg executes sync msg, and send tx to the destination blockchain
// the current function is only responsible for sending messages and is not responsible for processing data formats，
func (w *Writer) execToMapMsg(m msg.Message) bool {
	var (
		errorCount int64
		needNonce  = true
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			id, _ := m.Payload[0].(*big.Int)
			marshal, _ := m.Payload[1].([]byte)
			isEth2 := false
			if len(m.Payload) >= 3 {
				isEth2, _ = m.Payload[2].(bool)
			}

			method := mapprotocol.MethodUpdateBlockHeader
			if isEth2 {
				method = mapprotocol.MethodUpdateLightClient
			}

			err := w.toMap(m, id, marshal, method, needNonce)
			if err != nil {
				needNonce = w.needNonce(err)
				time.Sleep(constant.TxRetryInterval)
				errorCount++
				if errorCount >= 10 {
					util.Alarm(context.Background(), fmt.Sprintf("%s2map updateHeader failed, err is %s",
						mapprotocol.OnlineChaId[m.Source], err.Error()))
					errorCount = 0
				}
				continue
			}
			m.DoneCh <- struct{}{}
			return true
		}
	}
}

func (w *Writer) toMap(m msg.Message, id *big.Int, marshal []byte, method string, needNonce bool) error {
	err := w.conn.LockAndUpdateOpts(needNonce)
	if err != nil {
		w.log.Error("BlockToMap Failed to update nonce", "err", err)
		return err
	}

	data, err := mapprotocol.PackInput(mapprotocol.LightManger, method, id, marshal)
	if err != nil {
		w.log.Error("block2Map Failed to pack abi data", "err", err)
		w.conn.UnlockOpts()
		return err
	}
	fmt.Println("data ----------- ", "0x"+common.Bytes2Hex(data))
	tx, err := w.sendTx(&w.cfg.LightNode, nil, data)
	w.conn.UnlockOpts()
	if err == nil {
		// message successfully handled
		w.log.Info("Sync Header to map tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination,
			"method", method, "needNonce", needNonce, "nonce", w.conn.Opts().Nonce)
		err = w.txStatus(tx.Hash())
		if err != nil {
			w.log.Warn("TxHash Status is not successful, will retry", "err", err)
		} else {
			return nil
		}
	} else if w.cfg.SkipError {
		w.log.Warn("Execution failed, ignore this error, Continue to the next ", "err", err)
		return nil
	} else {
		for e := range constant.IgnoreError {
			if strings.Index(err.Error(), e) != -1 {
				w.log.Info("Ignore This Error, Continue to the next", "id", id, "method", method, "err", err)
				return nil
			}
		}
		w.log.Warn("Sync Header to map Execution failed, will retry", "id", id, "method", method, "err", err)
	}
	return err
}
