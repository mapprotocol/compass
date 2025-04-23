package sol

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"strings"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/constant"
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
	case msg.SwapSolProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var (
		errorCount int64
		log        = m.Payload[0].(*types.Log)
		//sign       = m.Payload[1].([][]byte)
		method = m.Payload[2].(string)
	)

	for {
		select {
		case <-w.stop:
			return false
		default:
			relayData, err := DecodeMessageRelay(log.Topics, common.Bytes2Hex(log.Data))
			if err != nil {
				w.log.Error("Error decoding relay data", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			w.log.Info("Relay data", "receiver", base58.Encode(relayData.Receiver), "dstToken", base58.Encode(relayData.DstToken), "outAmount", relayData.OutAmount)
			if relayData.Swap != nil {
				w.log.Info("Relay Swap data", "to", base58.Encode(relayData.Swap.ToToken), "receiver", base58.Encode(relayData.Swap.Receiver),
					"amount", relayData.Swap.MinAmount)
			}
			resp, err := w.solCrossIn(log, relayData)
			if err != nil {
				w.log.Error("Error in solCross in", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			txData := resp.Data[0].TxParam[0].Data

			w.log.Info("Send transaction", "srcHash", log.TxHash, "method", method)
			mcsTx, err := w.sendTx(txData)
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", log.TxHash, "mcsTx", mcsTx)
				err = w.txStatus(*mcsTx)
				if err == nil {
					w.log.Info("TxHash status is successful, will next tx")
					m.DoneCh <- struct{}{}
					return true
				}
			} else if w.cfg.SkipError && errorCount >= 9 {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "srcHash", log.TxHash, "err", err)
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
				w.log.Warn("Execution failed, will retry", "srcHash", log.TxHash, "err", err)
			}
			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(log.TxHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) sendTx(data string) (*solana.Signature, error) {
	w.log.Info("Sending transaction", "data", data)
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
	w.log.Info("Sending transaction", "blockHash", resp.Value.Blockhash)
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

	w.log.Info("Sending will transaction")
	sig, err := w.conn.cli.SendTransactionWithOpts(context.TODO(), trx, rpc.TransactionOpts{
		SkipPreflight: false,
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
		version := uint64(0)
		_, err := w.conn.cli.GetTransaction(context.Background(), txHash, &rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase58,
			Commitment:                     rpc.CommitmentFinalized,
			MaxSupportedTransactionVersion: &version,
		})
		if err != nil {
			count++
			w.log.Error("Failed to GetTransaction", "err", err)
			time.Sleep(5 * time.Second)
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			continue
		}

		txResult, err := w.conn.cli.GetSignatureStatuses(context.Background(), true, txHash)
		if err != nil {
			count++
			w.log.Error("Failed to GetSignatureStatuses", "err", err)
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("txResult ------------------ ", txResult)
		if txResult.Value == nil || len(txResult.Value) == 0 || txResult.Value[0].Err != nil {
			count++
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			continue
		}

		w.log.Info("Tx receipt status is success", "hash", txHash)
		return nil
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2sol failed, srcHash=%v err is %s", tx, err.Error()))
}

func (w *Writer) solCrossIn(l *types.Log, relayData *Relay) (*butter.SolCrossInResp, error) {
	signPri, _ := solana.PrivateKeyFromBase58(w.cfg.Pri)
	router := signPri.PublicKey().String()
	orderId := l.Topics[1]
	receiver := relayData.Receiver
	dstToken := relayData.DstToken
	minAmount := relayData.OutAmount
	if relayData.Swap != nil {
		receiver = relayData.Swap.Receiver
		minAmount = relayData.Swap.MinAmount
		dstToken = relayData.Swap.ToToken
	}
	query := fmt.Sprintf("tokenInAddress=%s&tokenAmount=%s&"+
		"tokenOutAddress=%s&"+
		"router=%s&minAmountOut=%s&orderIdHex=%s&receiver=%s&feeRatio=%s",
		base58.Encode(relayData.DstToken), relayData.OutAmount.String(),
		base58.Encode(dstToken),
		router, minAmount, orderId.Hex(), base58.Encode(receiver), "110",
	)
	data, err := butter.SolCrossIn(w.cfg.ButterHost, query)
	if err != nil {
		w.log.Error("Failed to butter.SolCrossIn", "err", err)
		return nil, err
	}

	return data, nil
}

type Resp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Tx       []string `json:"tx"`
		SwapData struct {
			ToToken        string `json:"to_token"`
			ToAddress      string `json:"to_address"`
			MinAmountOutBN string `json:"minAmountOutBN"`
		} `json:"swap_data"`
	} `json:"data"`
}
