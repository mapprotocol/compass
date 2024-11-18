package sol

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"strings"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

type Writer struct {
	cfg    *Config
	log    log15.Logger
	conn   *Connection
	stop   <-chan int
	sysErr chan<- error
}

func newWriter(conn *Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error) *Writer {
	return &Writer{
		cfg:    cfg,
		conn:   conn,
		log:    log,
		stop:   stop,
		sysErr: sysErr,
	}
}

func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination)
	switch m.Type {
	case msg.SwapWithMapProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var errorCount int64
	addr := w.cfg.McsContract[m.Idx]
	orderId32 := m.Payload[1].([32]byte)
	var orderId []byte
	for _, v := range orderId32 {
		orderId = append(orderId, v)
	}

	for {
		select {
		case <-w.stop:
			return false
		default:
			var inputHash interface{}
			if len(m.Payload) > 3 {
				inputHash = m.Payload[3]
			}
			method := m.Payload[4].(string)

			w.log.Info("Send transaction", "addr", addr, "srcHash", inputHash, "method", method)
			mcsTx, err := w.sendTx(addr, method, "")
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", inputHash, "mcsTx", mcsTx)
				err = w.txStatus(*mcsTx)
				if err != nil {
					w.log.Warn("TxHash status is not successful, will retry", "err", err)
				} else {
					m.DoneCh <- struct{}{}
					return true
				}
			} else if w.cfg.SkipError && errorCount >= 9 {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else {
				for e := range constant.IgnoreError {
					if strings.Index(err.Error(), e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "id", m.Destination, "err", err)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				w.log.Warn("Execution failed, will retry", "srcHash", inputHash, "err", err)
			}

			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(inputHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) sendTx(addr, method string, data string) (*solana.Signature, error) {
	bbs, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	trx, err := solana.TransactionFromBytes(bbs)
	if err != nil {
		return nil, err
	}
	resp, err := w.conn.cli.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	trx.Message.RecentBlockhash = resp.Value.Blockhash
	signPri, _ := solana.PrivateKeyFromBase58(w.cfg.Pri)
	// sign
	_, err = trx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key == signPri.PublicKey() {
			return &signPri
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	maxRetries := uint(5)
	minContextSlot := uint64(1)
	sig, err := w.conn.cli.SendTransactionWithOpts(context.TODO(), trx, rpc.TransactionOpts{
		SkipPreflight:  false,
		MaxRetries:     &maxRetries,
		MinContextSlot: &minContextSlot,
	})
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

func (w *Writer) txStatus(txHash solana.Signature) error {
	var count int64
	time.Sleep(time.Second * 2)
	for {
		result, err := w.conn.cli.GetTransaction(context.Background(), txHash, nil)
		if err != nil {
			w.log.Error("Failed to GetTransactionByID", "err", err)
			time.Sleep(constant.QueryRetryInterval)
			count++
			if count == 60 {
				return err
			}
			continue
		}
		if result.BlockTime.String() != "" {
			w.log.Info("Tx receipt status is success", "hash", txHash, "blockTIme", result.BlockTime.String())
			return nil
		}
		return nil
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2tron failed, srcHash=%v err is %s", tx, err.Error()))
}
