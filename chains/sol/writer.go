package sol

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"

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
		errorCount      int64
		receiveOpenDone bool
		log             = m.Payload[0].(*types.Log)
		method          = m.Payload[2].(string)
		sign            = m.Payload[3].([][]byte)
	)

	for {
		select {
		case <-w.stop:
			return false
		default:
			relayData, messageRelay, err := DecodeMessageRelay(log.Topics, common.Bytes2Hex(log.Data))
			if err != nil {
				w.log.Error("Error decoding relay data", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			w.log.Info("Relay data", "receiver", base58.Encode(relayData.Receiver), "dstToken", base58.Encode(relayData.DstToken), "outAmount", relayData.OutAmount)
			if relayData.Swap != nil {
				w.log.Info("Relay Swap data", "toToken", base58.Encode(relayData.Swap.ToToken), "receiver", base58.Encode(relayData.Swap.Receiver),
					"amount", relayData.Swap.MinAmount)
			}
			resp, err := w.solCrossIn(log, relayData, messageRelay, sign)
			if err != nil {
				w.log.Error("Error in solCross in", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			w.log.Info("Send transaction", "srcHash", log.TxHash, "method", method)
			mcsTxs, openDone, err := w.sendSolCrossInTxs(resp, receiveOpenDone)
			if openDone {
				receiveOpenDone = true
			}
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", log.TxHash, "mcsTxs", mcsTxs)
				m.DoneCh <- struct{}{}
				return true
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

func (w *Writer) sendSolCrossInTxs(resp *butter.SolCrossInResp, receiveOpenDone bool) ([]string, bool, error) {
	if resp == nil || len(resp.Data) == 0 {
		return nil, receiveOpenDone, errors.New("solCrossIn response data is empty")
	}
	txParams := solCrossInTxParams(resp)
	if len(txParams) == 0 {
		return nil, receiveOpenDone, errors.New("solCrossIn response txParam is empty")
	}

	if receiveOpenDone {
		execute, ok := findSolCrossInTxParam(txParams, "receiveExecute")
		if !ok {
			return nil, receiveOpenDone, errors.New("receiveOpen was already done, but receiveExecute txParam is missing")
		}
		txHash, err := w.sendSolCrossInTx(execute, 0)
		if err != nil {
			return nil, receiveOpenDone, err
		}
		return []string{txHash}, receiveOpenDone, nil
	}

	switch len(txParams) {
	case 1:
		txHash, err := w.sendSolCrossInTx(txParams[0], 0)
		if err != nil {
			return nil, receiveOpenDone, err
		}
		return []string{txHash}, receiveOpenDone, nil
	case 2:
		open, ok := findReceiveOpenTxParam(txParams)
		if !ok {
			return nil, receiveOpenDone, errors.New("receiveOpen txParam is missing")
		}
		execute, ok := findSolCrossInTxParam(txParams, "receiveExecute")
		if !ok {
			return nil, receiveOpenDone, errors.New("receiveExecute txParam is missing")
		}

		txHashes := make([]string, 0, 2)
		txHash, err := w.sendSolCrossInTx(open, 0)
		if err != nil {
			return nil, receiveOpenDone, err
		}
		receiveOpenDone = true
		txHashes = append(txHashes, txHash)

		txHash, err = w.sendSolCrossInTx(execute, 1)
		if err != nil {
			return txHashes, receiveOpenDone, err
		}
		txHashes = append(txHashes, txHash)
		return txHashes, receiveOpenDone, nil
	case 3:
		openExecute, ok := findSolCrossInTxParam(txParams, "receiveOpenExecute")
		if !ok {
			return nil, receiveOpenDone, errors.New("receiveOpenExecute txParam is missing")
		}
		txHash, err := w.sendSolCrossInTx(openExecute, 0)
		if err != nil {
			return nil, receiveOpenDone, err
		}
		return []string{txHash}, receiveOpenDone, nil
	default:
		return nil, receiveOpenDone, fmt.Errorf("unsupported solCrossIn txParam count: %d", len(txParams))
	}
}

func solCrossInTxParams(resp *butter.SolCrossInResp) []butter.SolCrossInTxParam {
	for _, item := range resp.Data {
		if len(item.TxParam) > 0 {
			return item.TxParam
		}
	}
	return nil
}

func findReceiveOpenTxParam(txParams []butter.SolCrossInTxParam) (butter.SolCrossInTxParam, bool) {
	if txParam, ok := findSolCrossInTxParam(txParams, "receiveOpen"); ok {
		return txParam, true
	}
	return findSolCrossInTxParam(txParams, "receiverOpen")
}

func findSolCrossInTxParam(txParams []butter.SolCrossInTxParam, step string) (butter.SolCrossInTxParam, bool) {
	for _, txParam := range txParams {
		if txParam.Step == step {
			return txParam, true
		}
	}
	return butter.SolCrossInTxParam{}, false
}

func (w *Writer) sendSolCrossInTx(txParam butter.SolCrossInTxParam, idx int) (string, error) {
	if txParam.Data == "" {
		return "", fmt.Errorf("solCrossIn txParam[%d] data is empty", idx)
	}
	w.log.Info("Send solCrossIn transaction", "index", idx, "step", txParam.Step, "isRefund", txParam.IsRefund, "chainId", txParam.ChainID)
	mcsTx, err := w.sendTx(txParam.Data)
	if err != nil {
		return "", err
	}
	w.log.Info("Submitted solCrossIn transaction", "index", idx, "step", txParam.Step, "mcsTx", mcsTx)
	if err = w.txStatus(*mcsTx); err != nil {
		return "", err
	}
	return mcsTx.String(), nil
}

func (w *Writer) sendTx(data string) (*solana.Signature, error) {
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
	signPri, err := solana.PrivateKeyFromBase58(w.cfg.Pri)
	if err != nil {
		return nil, err
	}
	signed := false
	_, err = trx.PartialSign(func(key solana.PublicKey) *solana.PrivateKey {
		if key == signPri.PublicKey() {
			signed = true
			return &signPri
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !signed {
		return nil, fmt.Errorf("router signer %s not required by transaction", signPri.PublicKey())
	}
	if err = ensureTransactionFullySigned(trx); err != nil {
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

func ensureTransactionFullySigned(trx *solana.Transaction) error {
	required := int(trx.Message.Header.NumRequiredSignatures)
	if len(trx.Signatures) < required {
		return fmt.Errorf("transaction signatures length %d less than required %d", len(trx.Signatures), required)
	}
	missing := make([]string, 0)
	for idx := 0; idx < required; idx++ {
		if trx.Signatures[idx].IsZero() {
			missing = append(missing, trx.Message.AccountKeys[idx].String())
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("transaction missing required signatures: %s", strings.Join(missing, ","))
	}
	return nil
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

func solanaAddressString(addr []byte) string {
	if len(addr) == 32 {
		return base58.Encode(addr)
	}
	return string(addr)
}

func (w *Writer) solCrossIn(l *types.Log, relayData *Relay, messageRelay *MessageRelay, sign [][]byte) (*butter.SolCrossInResp, error) {
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
		"router=%s&minAmountOut=%s&orderIdHex=%s&receiver=%s&feeRatio=%s&mapTxHash=%s",
		base58.Encode(relayData.DstToken), relayData.OutAmount.String(),
		base58.Encode(dstToken),
		router, minAmount, orderId.Hex(), solanaAddressString(receiver), "110", l.TxHash,
	)
	bodySign := make([]string, 0)
	for _, s := range sign {
		bodySign = append(bodySign, hex.EncodeToString(s))
	}
	body := map[string]interface{}{
		"relayProof": map[string]interface{}{
			"orderId":          common.Hash(messageRelay.OrderId).Hex(),
			"chainAndGasLimit": common.Bytes2Hex(proof.Completion(messageRelay.ChainAndGasLimit.Bytes(), 32)),
			"messagePayload":   common.Bytes2Hex(messageRelay.Payload),
			"blockNum":         common.Bytes2Hex(proof.Completion(proof.GenLogBlockNumber(big.NewInt(0).SetUint64(l.BlockNumber), l.TxIndex, l.Index).Bytes(), 32)),
			"signatures":       bodySign,
		},
	}
	data, err := butter.SolCrossIn(w.cfg.ButterHost, query, body)
	if err != nil {
		w.mosAlarm(query, err)
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
