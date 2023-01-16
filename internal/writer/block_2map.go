package writer

import (
	"context"
	"fmt"
	"github.com/mapprotocol/compass/pkg/util"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

// execToMapMsg executes sync msg, and send tx to the destination blockchain
// the current function is only responsible for sending messages and is not responsible for processing data formats，
func (w *Writer) execToMapMsg(m msg.Message) bool {
	var errorCount int64
	for {
		select {
		case <-w.stop:
			return false
		default:
			id, _ := m.Payload[0].(*big.Int)
			marshal, _ := m.Payload[1].([]byte)
			isEth2 := false
			// Eth2 exclusive process
			if len(m.Payload) >= 3 {
				isEth2, _ = m.Payload[2].(bool)
			}

			method := mapprotocol.MethodUpdateBlockHeader
			if isEth2 {
				method = mapprotocol.MethodUpdateLightClient
			}

			err := w.toMap(m, id, marshal, method)
			if err != nil {
				time.Sleep(constant.TxRetryInterval)
				errorCount++
				if errorCount == 10 {
					util.Alarm(context.Background(), fmt.Sprintf("writer other to map header failed, id=(%d), err is %s",
						id.Uint64(), err.Error()))
					errorCount = 0
				}
				continue
			}
			m.DoneCh <- struct{}{}
			return true
		}
	}
}

func (w *Writer) toMap(m msg.Message, id *big.Int, marshal []byte, method string) error {
	err := w.conn.LockAndUpdateOpts()
	if err != nil {
		w.log.Error("BlockToMap Failed to update nonce", "err", err)
		return err
	}
	// These store the gas limit and price before a transaction is sent for logging in case of a failure
	// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
	// save header data
	data, err := mapprotocol.PackInput(mapprotocol.LightManger, method, id, marshal)
	//data, err := mapprotocol.PackInput(mapprotocol.Bsc, method, marshal)
	if err != nil {
		w.log.Error("block2Map Failed to pack abi data", "err", err)
		w.conn.UnlockOpts()
		return err
	}
	tx, err := w.sendTx(&w.cfg.LightNode, nil, data)
	w.conn.UnlockOpts()
	if err == nil {
		// message successfully handled
		w.log.Info("Sync Header to map tx execution", "tx", tx.Hash(), "src", m.Source, "dst", m.Destination, "method", method)
		time.Sleep(time.Second * 2)
		err = w.txStatus(tx.Hash())
		if err != nil {
			w.log.Warn("TxHash Status is not successful, will retry", "err", err)
			return err
		} else {
			return nil
		}
	} else if strings.Index(err.Error(), constant.EthOrderExist) != -1 {
		w.log.Info(constant.EthOrderExistPrint, "id", id, "method", method, "err", err)
		return nil
	} else if strings.Index(err.Error(), constant.HeaderIsHave) != -1 {
		w.log.Info(constant.HeaderIsHavePrint, "id", id, "method", method, "err", err)
		return nil
	} else if strings.Index(err.Error(), constant.InvalidStartBlock) != -1 {
		w.log.Info(constant.InvalidStartBlockPrint, "id", id, "method", method, "err", err)
		return nil
	} else if strings.Index(err.Error(), constant.InvalidSyncBlock) != -1 {
		w.log.Info(constant.InvalidSyncBlockPrint, "id", id, "method", method, "err", err)
		return nil
	} else if err.Error() == constant.ErrNonceTooLow.Error() || err.Error() == constant.ErrTxUnderpriced.Error() {
		w.log.Error("Sync Header to map Nonce too low, will retry", "id", id, "method", method)
	} else if strings.Index(err.Error(), "EOF") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
		w.log.Error("Sync Header to map encounter EOF, will retry", "id", id, "method", method)
	} else if strings.Index(err.Error(), "max fee per gas less than block base fee") != -1 {
		w.log.Error("gas maybe less than base fee, will retry", "id", id, "method", method)
	} else if strings.Index(err.Error(), constant.NotEnoughGas) != -1 {
		w.log.Error(constant.NotEnoughGasPrint, "id", id, "method", method)
	} else {
		w.log.Warn("Sync Header to map Execution failed, header may already been synced", "id", id, "method", method, "err", err)
	}
	return err
}
