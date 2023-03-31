package near

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/action"
	"github.com/mapprotocol/near-api-go/pkg/types/hash"
)

const (
	MethodOfUpdateBlockHeader  = "update_block_header"
	MethodOfTransferIn         = "transfer_in"
	MethodOfSwapIn             = "swap_in"
	MethodOfVerifyReceiptProof = "verify_receipt_proof"
)

var (
	OrderIdIsUsed         = "the event with order id"
	OrderIdIsUsedFlag2    = "is used"
	VerifyRangeMatch      = "cannot get epoch record for block"
	VerifyRangeMatchFlag2 = "expected range"
)

var ignoreError = map[string]struct{}{
	"invalid to address":                                        {},
	"invalid to chain token address":                            {},
	"transfer in token failed, maybe TO account does not exist": {},
}

// exeSyncMapMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMapMsg(m msg.Message) bool {
	var errorCount int64
	for {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts(false)
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}

			txHash, err := w.sendTx(w.cfg.lightNode, MethodOfUpdateBlockHeader, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync MapHeader to Near tx execution", "tx", txHash.String(), "src", m.Source, "dst", m.Destination)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), "block header height is incorrect") != -1 {
				w.log.Error("The header may have been synchronized，Continue to execute the next header")
				m.DoneCh <- struct{}{}
				return true
			} else {
				w.log.Warn("Execution failed will retry", "err", err)
			}
			errorCount++
			if errorCount >= 10 {
				util.Alarm(context.Background(), fmt.Sprintf("map2Near updateHeader failed, err is %s", err.Error()))
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
	var errorCount int64
	var inputHash interface{}
	if len(m.Payload) > 3 {
		inputHash = m.Payload[3]
	}
	data := m.Payload[0].([]byte)

	for {
		// First request whether the orderId already exists
		if len(m.Payload) > 1 {
			orderId := m.Payload[1].([]byte)
			exits, err := w.checkOrderId(w.cfg.mcsContract, orderId)
			if err != nil {
				w.log.Error("check orderId exist failed ", "err", err, "orderId", common.Bytes2Hex(orderId))
			}
			if exits {
				w.log.Info("Mcs orderId has been processed, Skip this request", "orderId", common.Bytes2Hex(orderId))
				m.DoneCh <- struct{}{}
				return true
			}
		}

		md := make(map[string]interface{}, 0)
		_ = json.Unmarshal(data, &md)
		verify, err := json.Marshal(map[string]interface{}{
			"receipt_proof": md["receipt_proof"],
		})
		if err != nil {
			w.log.Error("Verify Execution failed, Will retry", "srcHash", inputHash, "err", err)
			return false
		}
		txHash, err := w.sendTx(w.cfg.mcsContract, MethodOfVerifyReceiptProof, verify)
		if err == nil {
			w.log.Info("Verify Success", "mcsTx", txHash.String(), "srcHash", inputHash)
			time.Sleep(time.Second)
			break
		} else {
			for e := range ignoreError {
				if strings.Index(err.Error(), e) != -1 {
					w.log.Info("Ignore This Error, Continue to the next", "method", MethodOfVerifyReceiptProof, "srcHash", inputHash, "err", err)
					m.DoneCh <- struct{}{}
					return true
				}
			}
			w.log.Warn("Verify Execution failed, Will retry", "srcHash", inputHash, "err", err)
			errorCount++
			if errorCount >= 10 {
				util.Alarm(context.Background(), fmt.Sprintf("map2Near mos(verify_receipt_proof) failed, srcHash=%v err is %s", inputHash, err.Error()))
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}

	errorCount = 0
	for {
		select {
		case <-w.stop:
			return false
		default:
			method := MethodOfTransferIn
			if m.Payload[4].(string) == mapprotocol.MethodOfSwapIn {
				method = MethodOfSwapIn
			}
			w.log.Info("Send transaction", "addr", w.cfg.mcsContract, "srcHash", inputHash, "method", method)
			txHash, err := w.sendTx(w.cfg.mcsContract, method, data)
			if err == nil {
				w.log.Info("Submitted cross tx execution", "mcsTx", txHash.String(), "srcHash", inputHash)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), OrderIdIsUsed) != -1 && strings.Index(err.Error(), OrderIdIsUsedFlag2) != -1 {
				w.log.Info("Order id is used, Continue to the next", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), VerifyRangeMatch) != -1 && strings.Index(err.Error(), VerifyRangeMatchFlag2) != -1 {
				abandon := w.resolveVerifyRangeError(m.Payload[2].(uint64), err)
				w.log.Error("The block where the transaction is located is no longer verifiable", "srcHash", inputHash, "abandon", abandon, "err", err)
				if abandon {
					m.DoneCh <- struct{}{}
					return true
				}
			} else if w.cfg.skipError {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else {
				for e := range ignoreError {
					if strings.Index(err.Error(), e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "method", method, "srcHash", inputHash, "err", err)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				w.log.Warn("Execution failed, tx may already be complete", "srcHash", inputHash, "err", err)
				errorCount++
				if errorCount >= 10 {
					util.Alarm(context.Background(), fmt.Sprintf("map2Near mos(%s) failed, srcHash=%v err is %s", method, inputHash, err.Error()))
					errorCount = 0
				}
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

// sendTx send tx to an address with value and input data
func (w *writer) sendTx(toAddress string, method string, input []byte) (hash.CryptoHash, error) {
	w.log.Info("sendTx", "toAddress", toAddress)
	ctx := client.ContextWithKeyPair(context.Background(), *w.conn.Keypair())
	b := types.Balance{}
	if method == MethodOfTransferIn || method == MethodOfSwapIn || method == MethodOfVerifyReceiptProof {
		b, _ = types.BalanceFromString(near.Deposit)
	}
	res, err := w.conn.Client().TransactionSendAwait(
		ctx,
		w.cfg.from,
		toAddress,
		[]action.Action{
			action.NewFunctionCall(method, input, near.NewFunctionCallGas, b),
		},
		client.WithLatestBlock(),
		client.WithKeyPair(*w.conn.Keypair()),
	)
	if err != nil {
		return hash.CryptoHash{}, fmt.Errorf("failed to do txn: %w", err)
	}
	w.log.Debug("sendTx success", "res", res)
	if len(res.Status.Failure) != 0 {
		return hash.CryptoHash{}, fmt.Errorf("%s", string(res.Status.Failure))
	}
	return res.Transaction.Hash, nil
}

func (w *writer) checkOrderId(toAddress string, input []byte) (bool, error) {
	var fixedOrderId [32]byte
	for idx, v := range input {
		fixedOrderId[idx] = v
	}
	m := map[string]interface{}{
		"order_id": fixedOrderId,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return false, err
	}
	ctx := client.ContextWithKeyPair(context.Background(), *w.conn.Keypair())
	res, err := w.conn.Client().ContractViewCallFunction(ctx, toAddress, mapprotocol.MethodOfIsUsedEvent,
		base64.StdEncoding.EncodeToString(data), block.FinalityFinal())
	if err != nil {
		return false, fmt.Errorf("checkOrderId ContractViewCallFunction failed: %w", err)
	}
	var exist bool
	err = json.Unmarshal(res.Result, &exist)
	if err != nil {
		return false, err
	}
	return exist, nil
}

func (w *writer) resolveVerifyRangeError(currentHeight uint64, par error) (isAbandon bool) {
	var entityError Error
	err := json.Unmarshal([]byte(par.Error()), &entityError)
	if err != nil {
		w.log.Warn("near mcs back err is not appoint json format", "err", par)
		return
	}
	leftIdx := strings.Index(entityError.ActionError.Kind.FunctionCallError.ExecutionError, "[")
	rightIdx := strings.Index(entityError.ActionError.Kind.FunctionCallError.ExecutionError, "]")
	rangeStr := entityError.ActionError.Kind.FunctionCallError.ExecutionError[leftIdx:rightIdx]
	verifyRange := strings.Split(strings.TrimSpace(rangeStr), ",")
	if len(verifyRange) != 2 {
		w.log.Warn("near mcs back err is not appoint json format", "err", par)
		return
	}
	left, err := strconv.ParseInt(strings.TrimSpace(verifyRange[0]), 10, 64)
	if err != nil {
		w.log.Warn("left range resolve failed", "str", verifyRange[0], "err", par)
		return
	}
	right, err := strconv.ParseInt(strings.TrimSpace(verifyRange[1]), 10, 64)
	if err != nil {
		w.log.Warn("right range resolve failed", "str", verifyRange[1], "err", par)
		return
	}
	if currentHeight < uint64(left) {
		isAbandon = true
		return
	}
	if currentHeight > uint64(right) {
		time.Sleep(time.Minute * 2)
	}
	return
}

type Error struct {
	ActionError ActionError `json:"ActionError"`
}

type ActionError struct {
	Index int  `json:"index"`
	Kind  Kind `json:"kind"`
}

type Kind struct {
	FunctionCallError FunctionCallError `json:"FunctionCallError"`
}

type FunctionCallError struct {
	ExecutionError string `json:"ExecutionError"`
}
