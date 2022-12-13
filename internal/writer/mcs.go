package writer

import (
	"context"
	"fmt"
	"github.com/mapprotocol/compass/internal/constant"
	"math/big"
	"strings"
	"time"

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
	return w.callContractWithMsg(w.cfg.McsContract, m)
}

// callContractWithMsg contract using address and function signature with message info
func (w *Writer) callContractWithMsg(addr common.Address, m msg.Message) bool {
	for {
		select {
		case <-w.stop:
			return false
		default:
			// 先请求下orderId是否已经存在
			if len(m.Payload) > 1 {
				orderId := m.Payload[1].([]byte)
				exits, err := w.checkOrderId(&addr, orderId, mapprotocol.Mcs, mapprotocol.MethodOfOrderList)
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
			w.conn.UnlockOpts()

			var inputHash interface{}
			if len(m.Payload) > 3 {
				inputHash = m.Payload[3]
			}
			w.log.Info("send transaction", "addr", addr, "srcHash", inputHash)
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice
			mcsTx, err := w.sendMcsTx(&addr, nil, m.Payload[0].([]byte))
			//err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.LightManger, mapprotocol.MethodVerifyProofData)
			if err == nil {
				// message successfully handled]
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", inputHash,
					"mcsTx", mcsTx.Hash())
				// Query transaction status
				err = w.txStatus(mcsTx.Hash())
				if err != nil {
					w.log.Warn("TxHash Status is not successful, will retry", "err", err)
				} else {
					m.DoneCh <- struct{}{}
					return true
				}
			} else if strings.Index(err.Error(), constant.EthOrderExist) != -1 {
				w.log.Info(constant.EthOrderExistPrint, "srcHash", inputHash)
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == constant.ErrNonceTooLow.Error() || err.Error() == constant.ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry", "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), "EOF") != -1 { // When requesting the lightNode to return EOF, it indicates that there may be a problem with the network and it needs to be retried
				w.log.Error("Mcs encounter EOF, will retry", "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.NotEnoughGas) != -1 {
				w.log.Error(constant.NotEnoughGasPrint, "srcHash", inputHash, "err", err)
			} else if w.cfg.SkipError {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "srcHash", inputHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else if strings.Index(err.Error(), constant.NotPerMission) != -1 {
				w.log.Error(constant.NotPerMissionPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.AddressIsZero) != -1 {
				w.log.Error(constant.AddressIsZeroPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.VaultNotRegister) != -1 {
				w.log.Error(constant.VaultNotRegisterPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.InvalidVaultToken) != -1 {
				w.log.Error(constant.InvalidVaultTokenPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.InvalidMosContract) != -1 {
				w.log.Error(constant.InvalidMosContractPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.InvalidChainId) != -1 {
				w.log.Error(constant.InvalidChainIdPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.MapTokenNotRegistered) != -1 {
				w.log.Error(constant.MapTokenNotRegisteredPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.OutTokenNotRegistered) != -1 {
				w.log.Error(constant.OutTokenNotRegisteredPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.BalanceTooLow) != -1 {
				w.log.Error(constant.BalanceTooLowPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.VaultTokenNotRegistered) != -1 {
				w.log.Error(constant.VaultTokenNotRegisteredPrint, "srcHash", inputHash, "err", err)
			} else if strings.Index(err.Error(), constant.ChainTypeError) != -1 {
				w.log.Error(constant.ChainTypeErrorPrint, "srcHash", inputHash, "err", err)
			} else {
				w.log.Warn("Execution failed, will retry", "srcHash", inputHash, "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
	//w.log.Error("Submission of Execute transaction failed", "source", m.Source, "dest", m.Destination,
	//	"depositNonce", m.DepositNonce)
	//w.sysErr <- constant.ErrFatalTx
	//return false
}

func (w *Writer) call(toAddress *common.Address, input []byte, useAbi abi.ABI, method string) error {
	from := w.conn.Keypair().CommonAddress()
	outPut, err := w.conn.Client().CallContract(context.Background(),
		ethereum.CallMsg{
			From: from,
			To:   toAddress,
			Data: input,
		},
		nil,
	)
	if err != nil {
		w.log.Error("mcs callContract failed", "err", err.Error())
		return err
	}

	resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
	if err != nil {
		w.log.Error("proof call failed ", "err", err.Error())
		return err
	}

	ret := struct {
		Success bool
		Message string
		Logs    []byte
	}{}

	err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
	if err != nil {
		return errors.Wrap(err, "proof copy failed")
	}

	if !ret.Success {
		return fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
	}
	if ret.Success == true {
		w.log.Info("mcs verify log success", "success", ret.Success)
		//tmp, _ := rlp.EncodeToBytes(ret.Logs)
		w.log.Info("mcs verify log success", "logs", "0x"+common.Bytes2Hex(ret.Logs))
	}

	return nil
}

// sendTx send tx to an address with value and input data
func (w *Writer) sendMcsTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
	gasPrice := w.conn.Opts().GasPrice
	nonce := w.conn.Opts().Nonce
	from := w.conn.Keypair().CommonAddress()

	msg := ethereum.CallMsg{
		From:     from,
		To:       toAddress,
		GasPrice: gasPrice,
		Value:    value,
		Data:     input,
	}
	w.log.Debug("eth CallMsg", "msg", msg)
	gasLimit, err := w.conn.Client().EstimateGas(context.Background(), msg)
	if err != nil {
		w.log.Error("EstimateGas failed", "error:", err.Error())
		return nil, err
	}

	// td interface
	var td types.TxData
	// EIP-1559
	if gasPrice != nil {
		// legacy branch
		td = &types.LegacyTx{
			Nonce:    nonce.Uint64(),
			Value:    value,
			To:       toAddress,
			Gas:      gasLimit,
			GasPrice: gasPrice,
			Data:     input,
		}
	} else {
		// london branch
		td = &types.DynamicFeeTx{
			Nonce:     nonce.Uint64(),
			Value:     value,
			To:        toAddress,
			Gas:       gasLimit,
			GasTipCap: w.conn.Opts().GasTipCap,
			GasFeeCap: w.conn.Opts().GasFeeCap,
			Data:      input,
		}
	}

	tx := types.NewTx(td)
	chainID := big.NewInt(int64(w.cfg.Id))
	privateKey := w.conn.Keypair().PrivateKey()

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), privateKey)
	if err != nil {
		w.log.Error("SignTx failed", "error:", err.Error())
		return nil, err
	}

	err = w.conn.Client().SendTransaction(context.Background(), signedTx)
	if err != nil {
		w.log.Error("SendTransaction failed", "error:", err.Error())
		return nil, err
	}
	return signedTx, nil
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
	from := w.conn.Keypair().CommonAddress()
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
	time.Sleep(time.Second * 2)
	for {
		receipt, err := w.conn.Client().TransactionReceipt(context.Background(), txHash)
		if err != nil {
			if strings.Index(err.Error(), "not found") != -1 {
				w.log.Info("Tx is temporary not found, please wait...", "tx", txHash)
				time.Sleep(time.Millisecond * 900)
				count++
				if count == 20 {
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
