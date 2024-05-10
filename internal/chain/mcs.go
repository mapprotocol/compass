package chain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/mapprotocol"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *Writer) exeSwapMsg(m msg.Message) bool {
	return w.callContractWithMsg(w.cfg.McsContract[m.Idx], m)
}

// callContractWithMsg contract using address and function signature with message info
func (w *Writer) callContractWithMsg(addr common.Address, m msg.Message) bool {
	var (
		errorCount, checkIdCount int64
		needNonce                = true
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			orderId := m.Payload[1].([]byte)
			exits, err := w.checkOrderId(&addr, orderId, mapprotocol.Mcs, mapprotocol.MethodOfOrderList)
			if err != nil {
				w.log.Error("check orderId exist failed ", "err", err, "orderId", common.Bytes2Hex(orderId))
				checkIdCount++
				if checkIdCount == 10 {
					util.Alarm(context.Background(), fmt.Sprintf("writer mos checkOrderId failed, err is %s", err.Error()))
					checkIdCount = 0
				}
			}
			if exits {
				w.log.Info("Mcs orderId has been processed, Skip this request", "orderId", common.Bytes2Hex(orderId))
				m.DoneCh <- struct{}{}
				return true
			}

			err = w.conn.LockAndUpdateOpts(needNonce)
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			var inputHash interface{}
			if len(m.Payload) > 3 {
				inputHash = m.Payload[3]
			}
			w.log.Info("Send transaction", "addr", addr, "srcHash", inputHash, "needNonce", needNonce, "nonce", w.conn.Opts().Nonce)
			mcsTx, err := w.sendTx(&addr, nil, m.Payload[0].([]byte))
			//err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.Other, mapprotocol.MethodVerifyProofData)
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", inputHash, "mcsTx", mcsTx.Hash())
				err = w.txStatus(mcsTx.Hash())
				if err != nil {
					w.log.Warn("TxHash Status is not successful, will retry", "err", err)
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
			needNonce = w.needNonce(err)
			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(m, inputHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) merlinWithMsg(m msg.Message) bool {
	var (
		errorCount int64
		needNonce  = true
		addr       = w.cfg.McsContract[m.Idx]
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts(needNonce)
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			var inputHash = m.Payload[3]
			w.log.Info("Send transaction", "method", m.Payload[4], "srcHash", inputHash, "needNonce", needNonce, "nonce", w.conn.Opts().Nonce)
			mcsTx, err := w.sendTx(&addr, nil, m.Payload[0].([]byte))
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", inputHash, "mcsTx", mcsTx.Hash())
				err = w.txStatus(mcsTx.Hash())
				if err != nil {
					w.log.Warn("Store TxHash Status is not successful, will retry", "err", err)
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
				w.log.Warn("Execution SwapInVerify failed, will retry", "srcHash", inputHash, "err", err)
			}

			needNonce = w.needNonce(err)
			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(m, inputHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) mosAlarm(m msg.Message, tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos %s2%s failed, srcHash=%v err is %s", mapprotocol.OnlineChaId[m.Source],
		mapprotocol.OnlineChaId[m.Destination], tx, err.Error()))
}

func (w *Writer) call(toAddress *common.Address, input []byte, useAbi abi.ABI, method string, ret interface{}) error {
	from := w.conn.Keypair().Address
	outPut, err := w.conn.Client().CallContract(context.Background(),
		ethereum.CallMsg{
			From: from,
			To:   toAddress,
			Data: input,
		},
		nil,
	)
	if err != nil {
		w.log.Error("Mcs callContract verify failed", "err", err.Error())
		return err
	}

	resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
	if err != nil {
		w.log.Error("Writer Unpack failed ", "method", method, "err", err.Error())
		return err
	}

	err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
	if err != nil {
		return errors.Wrap(err, "resp copy failed")
	}

	return nil
}

func (w *Writer) checkOrderId(toAddress *common.Address, input []byte, useAbi abi.ABI, method string) (bool, error) {
	var fixedOrderId [32]byte
	for idx, v := range input {
		fixedOrderId[idx] = v
	}
	data, err := mapprotocol.PackInput(useAbi, method, fixedOrderId)
	if err != nil {
		return false, err
	}
	from := w.conn.Keypair().Address
	outPut, err := w.conn.Client().CallContract(context.Background(),
		ethereum.CallMsg{
			From: from,
			To:   toAddress,
			Data: data,
		},
		nil,
	)
	if err != nil {
		return false, errors.Wrap(err, "callContract failed")
	}

	resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
	if err != nil {
		return false, errors.Wrap(err, "output Unpack failed")
	}

	var exist bool
	err = useAbi.Methods[method].Outputs.Copy(&exist, resp)
	if err != nil {
		return false, errors.Wrap(err, "checkOrderId output copy failed")
	}

	return exist, nil
}

func (w *Writer) txStatus(txHash common.Hash) error {
	var count int64
	//time.Sleep(time.Second * 2)
	for {
		_, pending, err := w.conn.Client().TransactionByHash(context.Background(), txHash) // Query whether it is on the chain
		if pending {
			w.log.Info("Tx is Pending, please wait...", "tx", txHash)
			time.Sleep(w.queryInterval())
			count++
			if count == 60 {
				return errors.New("The Tx pending state is too long")
			}
			continue
		}
		if err != nil {
			time.Sleep(w.queryInterval())
			count++
			if count == 60 {
				return err
			}
			w.log.Error("Tx Found failed, please wait...", "tx", txHash, "err", err)
			continue
		}
		break
	}
	count = 0
	for {
		receipt, err := w.conn.Client().TransactionReceipt(context.Background(), txHash) // Query receipt after chaining
		if err != nil {
			if strings.Index(err.Error(), "not found") != -1 {
				w.log.Info("Tx is temporary not found, please wait...", "tx", txHash)
				time.Sleep(w.queryInterval())
				count++
				if count == 40 {
					return err
				}
				continue
			}
			return err
		}

		if receipt.Status == types.ReceiptStatusSuccessful {
			w.log.Info("Tx receipt status is success", "hash", txHash)
			return nil
		}
		return fmt.Errorf("txHash(%s), status not success, current status is (%d)", txHash, receipt.Status)
	}
}

func (w *Writer) queryInterval() time.Duration {
	switch w.cfg.Id {
	case 22776:
		return time.Second * 3
	default:
		return constant.QueryRetryInterval
	}
}
