package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

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
	//return w.callContractWithMsg(w.cfg.LightNode, m)
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
			orderId := m.Payload[1].([32]byte)
			exits, err := w.checkOrderId(&addr, orderId, mapprotocol.Mcs, mapprotocol.MethodOfOrderList)
			if err != nil {
				w.log.Error("check orderId exist failed ", "err", err)
				checkIdCount++
				if checkIdCount == 10 {
					util.Alarm(context.Background(), fmt.Sprintf("writer mos checkOrderId failed, err is %s", err.Error()))
					checkIdCount = 0
				}
			}
			if exits {
				w.log.Info("Mcs orderId has been processed, Skip this request", "orderId", orderId)
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
			//err := w.call(&addr, m.Payload[0].([]byte), mapprotocol.Other, mapprotocol.MethodVerifyProofData)
			mcsTx, err := w.sendTx(&addr, nil, m.Payload[0].([]byte))
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination,
					"srcHash", inputHash, "mcsTx", mcsTx.Hash())
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
					w.log.Info("Success idx ", "src", inputHash, "idx", constant.MapLogIdx[inputHash.(common.Hash).Hex()], "orderId", constant.MapOrderId[inputHash.(common.Hash).Hex()])
					constant.MapLogIdx[mcsTx.Hash().Hex()] = constant.MapLogIdx[inputHash.(common.Hash).Hex()]
					constant.MapOrderId[mcsTx.Hash().Hex()] = constant.MapOrderId[inputHash.(common.Hash).Hex()]
					w.log.Info("Success idx ", "des", mcsTx, "idx", constant.MapLogIdx[mcsTx.Hash().Hex()])
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

func (w *Writer) proposal(m msg.Message) bool {
	var (
		errorCount int64
		needNonce  = true
		addr       = w.cfg.OracleNode
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			pack := m.Payload[0].([]byte)
			receiptHash := m.Payload[1].(*common.Hash)
			blockNumber := m.Payload[2].(*big.Int)
			hash := common.Bytes2Hex(crypto.Keccak256(pack))
			sign, err := personalSign(string(common.Hex2Bytes(hash)), w.conn.Keypair().PrivateKey)
			if err != nil {
				return false
			}
			var fixedHash [32]byte
			for i, v := range receiptHash {
				fixedHash[i] = v
			}

			data, err := mapprotocol.SignerAbi.Pack(mapprotocol.MethodOfPropose, big.NewInt(int64(m.Source)), blockNumber, fixedHash, sign)
			if err != nil {
				return false
			}

			err = w.conn.LockAndUpdateOpts(needNonce)
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			w.log.Info("Send tronProposal transaction", "addr", addr, "needNonce", needNonce, "nonce", w.conn.Opts().Nonce)
			mcsTx, err := w.sendTx(&addr, nil, data)
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "mcsTx", mcsTx.Hash())
				err = w.txStatus(mcsTx.Hash())
				if err != nil {
					w.log.Warn("Store TxHash Status is not successful, will retry", "err", err)
				} else {
					m.DoneCh <- struct{}{}
					return true
				}
			} else if w.cfg.SkipError && errorCount >= 9 {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "err", err)
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
				w.log.Warn("Execution SwapInVerify failed, will retry", "err", err)
			}

			needNonce = w.needNonce(err)
			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(m, "proposal", err)
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

func (w *Writer) call(toAddress *common.Address, input []byte, useAbi abi.ABI, method string) error {
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

	ret := struct {
		Success bool
		Message string
		Log     interface{}
	}{}

	err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
	if err != nil {
		return errors.Wrap(err, "resp copy failed")
	}

	if !ret.Success {
		return fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
	}
	if ret.Success == true {
		w.log.Info("Mcs verify log success", "success", ret.Success)
		w.log.Info("Mcs verify log success", "data", resp)
	}

	return nil
}

func (w *Writer) checkOrderId(toAddress *common.Address, input [32]byte, useAbi abi.ABI, method string) (bool, error) {
	data, err := mapprotocol.PackInput(useAbi, method, input)
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
		pending, err := w.conn.Client().IsPendingByTxHash(context.Background(), txHash) // Query whether it is on the chain
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
