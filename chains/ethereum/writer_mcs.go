package ethereum

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

// exeSwapMsg executes swap msg, and send tx to the destination blockchain
func (w *writer) exeSwapMsg(m msg.Message) bool {
	//return w.callContractWithMsg(w.cfg.bridgeContract, m)
	return w.callContractWithMsg(mapprotocol.Eth2MapTmpAddress, m) // local test eth -> map
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
			//err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.Eth2MapTransferInAbi, mapprotocol.MethodOfTransferIn)
			_, err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.Verify, mapprotocol.MethodVerifyProofData)
			fmt.Println("第一次触发, err is ", err)
			time.Sleep(3 * time.Second)

			//if err == nil {
			//	input, err := mapprotocol.PackDecodeInput(mapprotocol.MethodOfDecodeTxLog, logs)
			//	if err != nil {
			//		fmt.Println("第一次触发 decode log, pack input failed , err is ", err)
			//		return false
			//	}
			//	_, err = w.call(&w.cfg.bridgeContract, input, mapprotocol.Decode, mapprotocol.MethodOfDecodeTxLog)
			//	fmt.Println("第一次触发 decode log, err is ", err)
			//	if err != nil {
			//		return false
			//	}
			//}
			//err = w.call(&addr, m.Payload[0].([]byte), mapprotocol.Eth2MapTransferInAbi, mapprotocol.MethodOfTransferIn)
			//fmt.Println("第二次触发, err is ", nil)
			w.conn.UnlockOpts()

			if err == nil {
				// message successfully handled
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce)
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

func (w *writer) call(toAddress *common.Address, input []byte, useAbi abi.ABI, method string) ([]byte, error) {
	from := w.conn.Keypair().CommonAddress()
	output, err := w.conn.Client().CallContract(context.Background(),
		ethereum.CallMsg{
			From: from,
			To:   toAddress,
			Data: input,
		},
		nil,
	)
	if err != nil {
		w.log.Error("mcs callContract failed", "err", err.Error())
		return nil, err
	}

	resp, err := useAbi.Methods[method].Outputs.Unpack(output)
	if err != nil {
		w.log.Error("proof call failed ", "err", err.Error())
		return nil, err
	}

	if method == mapprotocol.MethodOfDecodeTxLog {
		//ret := struct {
		//	Address common.Address
		//	Topics  []byte
		//	Data    []byte
		//}

		txLog := make([]struct {
			Address common.Address
			Topics  []byte
			Data    []byte
		}, 0)
		err = useAbi.Methods[method].Outputs.Copy(&txLog, resp)
		if err != nil {
			return nil, errors.Wrap(err, "decode copy failed")
		}

		fmt.Println("ret -------------", txLog)
		return nil, nil
	}
	ret := struct {
		Success bool
		Message string
		Logs    []byte
	}{}

	w.log.Info("verify ", "back resp len", len(resp), "resp", resp)
	err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
	if err != nil {
		return nil, errors.Wrap(err, "proof copy failed")
	}
	if !ret.Success {
		return nil, fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
	}
	if ret.Success == true {
		w.log.Info("mcs verify log success", "success", ret.Success, "msg", ret.Message)
		tmp, _ := rlp.EncodeToBytes(ret.Logs)
		w.log.Info("mcs verify log success", "logs", common.Bytes2Hex(tmp))
		// 发送到map的合约，验证log
	}

	return ret.Logs, nil
}