package ethereum

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/msg"
)

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
	return w.callContractWithMsg(w.cfg.bridgeContract, m)
	//return w.callContractWithMsg(mapprotocol.Eth2MapTmpAddress, m) // local test eth -> map
}

// callContractWithMsg call contract using address and function signature with message info
func (w *writer) callContractWithMsg(addr common.Address, m msg.Message) bool {
	for i := 0; i < TxRetryLimit; i++ {
		select {
		case <-w.stop:
			return false
		default:
			err := w.conn.LockAndUpdateOpts()
			if err != nil {
				w.log.Error("Failed to update nonce", "err", err)
				return false
			}
			// These store the gas limit and price before a transaction is sent for logging in case of a failure
			// This is necessary as tx will be nil in the case of an error when sending VoteProposal()
			gasLimit := w.conn.Opts().GasLimit
			gasPrice := w.conn.Opts().GasPrice
			mcsTx, err := w.sendMcsTx(&addr, nil, m.Payload[0].([]byte))
			//err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.Verify, mapprotocol.MethodVerifyProofData)
			w.log.Info("send transaction", "addr", addr)
			w.conn.UnlockOpts()

			if err == nil {
				// message successfully handled
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce, "mcsTx", mcsTx.Hash())
				m.DoneCh <- struct{}{}
				return true
			} else if err.Error() == ErrNonceTooLow.Error() || err.Error() == ErrTxUnderpriced.Error() {
				w.log.Error("Nonce too low, will retry")
				time.Sleep(TxRetryInterval)
			} else {
				w.log.Warn("Execution failed, tx may already be complete", "gasLimit", gasLimit, "gasPrice", gasPrice, "err", err)
				time.Sleep(TxRetryInterval)
			}
		}
	}
	w.log.Error("Submission of Execute transaction failed", "source", m.Source, "dest", m.Destination, "depositNonce", m.DepositNonce)
	w.sysErr <- ErrFatalTx
	return false
}

//func (w *writer) call(toAddress *common.Address, input []byte, useAbi abi.ABI, method string) error {
//	from := w.conn.Keypair().CommonAddress()
//	outPut, err := w.conn.Client().CallContract(context.Background(),
//		ethereum.CallMsg{
//			From: from,
//			To:   toAddress,
//			Data: input,
//		},
//		nil,
//	)
//	if err != nil {
//		w.log.Error("mcs callContract failed", "err", err.Error())
//		return err
//	}
//
//	resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
//	if err != nil {
//		w.log.Error("proof call failed ", "err", err.Error())
//		return err
//	}
//
//	ret := struct {
//		Success bool
//		Message string
//		Logs    []byte
//	}{}
//
//	w.log.Info("verify ", "back resp len", len(resp), "resp", resp)
//	err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
//	if err != nil {
//		return errors.Wrap(err, "proof copy failed")
//	}
//	if !ret.Success {
//		return fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
//	}
//	if ret.Success == true {
//		w.log.Info("mcs verify log success", "success", ret.Success)
//		//tmp, _ := rlp.EncodeToBytes(ret.Logs)
//		w.log.Info("mcs verify log success", "logs", "0x"+common.Bytes2Hex(ret.Logs))
//	}
//
//	return nil
//}

// sendTx send tx to an address with value and input data
func (w *writer) sendMcsTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
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
	chainID := big.NewInt(int64(w.cfg.id))
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
