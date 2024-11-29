package sol

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/pkg/errors"
	"io"
	"math/big"
	"net/http"
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
	case msg.SwapSolProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

type GenerateRequest struct {
	LogAddr      string   `json:"logAddr"`
	LogTopics    []string `json:"logTopics"`
	LogData      string   `json:"logData"`
	Signatures   []string `json:"signatures"`
	OraclePacked string   `json:"oraclePacked"`
	Relayer      string   `json:"relayer"`
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var errorCount int64
	addr := w.cfg.McsContract[m.Idx]
	log := m.Payload[0].(*types.Log)
	sign := m.Payload[1].([][]byte)
	method := m.Payload[2].(string)

	for {
		select {
		case <-w.stop:
			return false
		default:
			// 构建http request
			mulInfo, err := chain.MulSignInfo(0, uint64(w.cfg.MapChainID))
			if err != nil {
				w.log.Error("Failed to mul sign info", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			receipt, err := chain.GenLogReceipt(log)
			if err != nil {
				w.log.Error("Failed to generate receipt", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			pack, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receipt,
				mulInfo.Version, big.NewInt(0).SetUint64(log.BlockNumber), big.NewInt(int64(w.cfg.MapChainID)))
			if err != nil {
				w.log.Error("Failed to pack abi", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}

			topic := make([]string, 0)
			for _, ele := range log.Topics {
				topic = append(topic, ele.String())
			}
			signs := make([]string, 0)
			for _, ele := range sign {
				signs = append(signs, "0x"+common.Bytes2Hex(ele))
			}
			request := GenerateRequest{
				LogAddr:      log.Address.String(),
				LogTopics:    topic,
				LogData:      "0x" + common.Bytes2Hex(log.Data),
				Signatures:   signs,
				OraclePacked: "0x" + common.Bytes2Hex(pack),
				Relayer:      "G4UPgaqx8ZWm5NZWee5PEX3RYx6kTUHiHphpWjJhARB5",
			}
			reqData, _ := json.Marshal(&request)
			resp, err := http.Post("http://localhost:3000/message_in", "application/json", bytes.NewBuffer(reqData))
			if err != nil {
				w.log.Error("Failed to send message", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			w.log.Info("Send transaction", "addr", addr, "srcHash", log.TxHash, "method", method)
			defer resp.Body.Close()
			all, err := io.ReadAll(resp.Body)
			if err != nil {
				w.log.Error("Failed to read response body", "err", err)
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			w.log.Info("Request back data", "body", string(all))

			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", log.TxHash, "mcsTx", "")
				err = w.txStatus(solana.Signature{})
				if err != nil {
					w.log.Warn("Store TxHash Status is not successful, will retry", "err", err)
				} else {
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
				w.log.Warn("Execution SwapInVerify failed, will retry", "srcHash", log.TxHash, "err", err)
			}

			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(log.TxHash.String(), err)
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
		txResult, err := w.conn.cli.GetSignatureStatuses(context.Background(), true, txHash)
		if err != nil {
			w.log.Error("Failed to GetSignatureStatuses", "err", err)
			count++
			if count >= 10 {
				return errors.New("The Tx maybe not found")
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if txResult.Value == nil || len(txResult.Value) == 0 || txResult.Value[0].Err != nil {
			count++
			continue
		}

		w.log.Info("Tx receipt status is success", "hash", txHash)
		return nil
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2tron failed, srcHash=%v err is %s", tx, err.Error()))
}
