package near

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"strconv"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/near"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/action"
	"github.com/mapprotocol/near-api-go/pkg/types/hash"
)

const (
	AbiMethodOfUpdateBlockHeader = "update_block_header"
	AbiMethodOfTransferIn        = "transfer_in"
	AbiMethodOfSwapIn            = "swap_in"
)

var (
	ErrNonceTooLow   = errors.New("nonce too low")
	ErrFatalTx       = errors.New("submission of transaction failed")
	ErrTxUnderpriced = errors.New("replacement transaction underpriced")
)

var (
	OrderIdIsUsed         = "the event with order id"
	OrderIdIsUsedFlag2    = "is used"
	ToAddressError        = "invalid to address"
	ValidChainToken       = "invalid to chain token address"
	TransferInToken       = "transfer in token failed, maybe TO account does not exist"
	VerifyRangeMatch      = "cannot get epoch record for block"
	VerifyRangeMatchFlag2 = "expected range"
	TokenNotSupport       = "to_chain_token"
	TokenNotSupportFlag2  = "is not mcs token or fungible token or native token"
	TokenFailed           = "transfer in token failed"
	WithdrawFailed        = "near withdraw failed"
)

// exeSyncMapMsg executes sync msg, and send tx to the destination blockchain
func (w *writer) exeSyncMapMsg(m msg.Message) bool {
	for i := 0; i < constant.TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}

			txHash, err := w.sendTx(w.cfg.lightNode, AbiMethodOfUpdateBlockHeader, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Sync MapHeader to Near tx execution", "tx", txHash.String(), "src", m.Source, "dst", m.Destination)
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry", "err", err)
			} else if strings.Index(err.Error(), "EOF") != -1 || strings.Index(err.Error(), "unexpected end of JSON input") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Sync Header to map encounter EOF, will retry")
			} else if strings.Index(err.Error(), "block header height is incorrect") != -1 {
				w.log.Error("The header may have been synchronized，Continue to execute the next header")
				m.DoneCh <- struct{}{}
				return true
			} else {
				w.log.Warn("Execution failed will retry", "err", err)
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
	w.log.Error("Submission of Sync MapHeader transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
	for i := 0; i < constant.TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return false
		default:
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

			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}

			var inputHash interface{}
			if len(m.Payload) > 3 {
				inputHash = m.Payload[3]
			}
			method := AbiMethodOfTransferIn
			if m.Payload[4].(string) == mapprotocol.MethodOfSwapIn {
				method = AbiMethodOfSwapIn
			}
			w.log.Info("send transaction", "addr", w.cfg.mcsContract, "srcHash", inputHash, "method", method)
			// sendtx using general method
			txHash, err := w.sendTx(w.cfg.mcsContract, method, m.Payload[0].([]byte))
			w.conn.UnlockOpts()
			if err == nil {
				// message successfully handled
				w.log.Info("Submitted cross tx execution", "mcsTx", txHash.String(), "src", m.Source, "dst", m.Destination, "srcHash", inputHash)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), OrderIdIsUsed) != -1 && strings.Index(err.Error(), OrderIdIsUsedFlag2) != -1 {
				w.log.Info("Order id is used, Continue to the next", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), ToAddressError) != -1 {
				w.log.Info("Tx to address is error, Continue to the next", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), ValidChainToken) != -1 {
				w.log.Info("Tx have invalid to chain token address, Continue to the next", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), TransferInToken) != -1 {
				w.log.Info("Tx transfer in token failed, maybe TO account does not exist", "srcHash", inputHash, "err", err)
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
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry", "srcHash", inputHash)
			} else if strings.Index(err.Error(), "EOF") != -1 || strings.Index(err.Error(), "unexpected end of JSON input") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Mcs encounter EOF, will retry", "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), TokenNotSupport) != -1 && strings.Index(err.Error(), TokenNotSupportFlag2) != -1 {
				w.log.Error("Transfer Token is not supported", "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), TokenFailed) != -1 {
				w.log.Error("Insufficient vault balance of NEP141 Or The target user does not exist, Please check", "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), WithdrawFailed) != -1 {
				w.log.Error("Insufficient vault when native token is transferred in", "srcHash", inputHash, "err", err)
			} else {
				w.log.Warn("Execution failed, tx may already be complete", "srcHash", inputHash, "err", err)
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
	w.log.Error("Submission of Execute transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

// sendTx send tx to an address with value and input data
func (w *writer) sendTx(toAddress string, method string, input []byte) (hash.CryptoHash, error) {
	w.log.Info("sendTx", "toAddress", toAddress)
	ctx := client.ContextWithKeyPair(context.Background(), *w.conn.Keypair())
	b := types.Balance{}
	if method == AbiMethodOfTransferIn || method == AbiMethodOfSwapIn {
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
