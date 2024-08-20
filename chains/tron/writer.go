package tron

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lbtsm/gotron-sdk/pkg/proto/api"

	"github.com/lbtsm/gotron-sdk/pkg/store"

	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/pkg/errors"

	"github.com/lbtsm/gotron-sdk/pkg/proto/core"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lbtsm/gotron-sdk/pkg/client/transaction"
	"github.com/lbtsm/gotron-sdk/pkg/keystore"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

var multiple = big.NewInt(420)

type Writer struct {
	cfg    *Config
	log    log15.Logger
	conn   *Connection
	stop   <-chan int
	sysErr chan<- error
	pass   []byte
	ks     *keystore.KeyStore
	acc    *keystore.Account
}

func newWriter(conn *Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error, pass []byte) *Writer {
	return &Writer{
		cfg:    cfg,
		conn:   conn,
		log:    log,
		stop:   stop,
		sysErr: sysErr,
		pass:   pass,
	}
}

func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination)
	switch m.Type {
	case msg.SyncFromMap:
		return w.syncMapToTron(m)
	case msg.SwapWithMapProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) syncMapToTron(m msg.Message) bool {
	var (
		errorCount int64
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			input := m.Payload[0].([]byte)
			tx, err := w.sendTx(w.cfg.LightNode, mapprotocol.MethodUpdateBlockHeader, input, 0, 1, 0, false)
			if err == nil {
				w.log.Info("Sync Map Header to tron chain tx execution", "tx", tx, "src", m.Source, "dst", m.Destination)
				err = w.txStatus(tx)
				if err != nil {
					w.log.Warn("TxHash Status is not successful, will retry", "err", err)
				} else {
					m.DoneCh <- struct{}{}
					return true
				}
			} else if w.cfg.SkipError {
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
			}
			errorCount++
			if errorCount >= 10 {
				util.Alarm(context.Background(), fmt.Sprintf("map2tron updateHeader failed, err is %s", err.Error()))
				errorCount = 0
			}
			time.Sleep(constant.BalanceRetryInterval)
		}
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var errorCount, checkIdCount int64
	addr := w.cfg.McsContract[m.Idx]
	orderId32 := m.Payload[1].([32]byte)
	var orderId []byte
	for _, v := range orderId32 {
		orderId = append(orderId, v)
	}
	exits, err := w.checkOrderId(addr, orderId)
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

	for {
		select {
		case <-w.stop:
			return false
		default:
			var inputHash interface{}
			if len(m.Payload) > 3 {
				inputHash = m.Payload[3]
			}
			method := m.Payload[4].(string)

			contract, err := w.conn.cli.TriggerConstantContractByEstimate(w.cfg.From, addr, m.Payload[0].([]byte), 0)
			if err != nil {
				w.log.Error("Failed to TriggerConstantContract EstimateEnergy", "err", err)
				time.Sleep(time.Minute)
				continue
			}
			for _, v := range contract.ConstantResult {
				w.log.Info("Contract result", "err", string(v))
				ele := strings.TrimSpace(string(v))
				if ele == "" {
					continue
				}
				for e := range constant.IgnoreError {
					if strings.Index(ele, e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "inputHash", inputHash, "err", ele)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				err = errors.New(ele)
			}
			if err != nil {
				w.mosAlarm(inputHash, fmt.Errorf("contract result failed, err is %v", err))
				time.Sleep(time.Minute)
				continue
			}
			w.log.Info("Trigger Contract result detail", "used", contract.EnergyUsed, "method", method)
			time.Sleep(time.Minute * 10)

			err = w.rentEnergy(contract.EnergyUsed, method)
			if err != nil {
				w.log.Info("Check energy failed", "srcHash", inputHash, "err", err)
				w.mosAlarm(inputHash, errors.Wrap(err, "please admin handler"))
				time.Sleep(time.Minute * 5)
				continue
			}

			w.log.Info("Send transaction", "addr", addr, "srcHash", inputHash, "method", method)
			mcsTx, err := w.sendTx(addr, method, m.Payload[0].([]byte), 0, int64(w.cfg.GasMultiplier),
				0, false)
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", inputHash, "mcsTx", mcsTx)
				err = w.txStatus(mcsTx)
				if err != nil {
					w.log.Warn("TxHash status is not successful, will retry", "err", err)
					m.DoneCh <- struct{}{}
					return true
				} else {
					w.newReturn(method)
					w.log.Info("Success idx ", "src", inputHash, "idx", constant.MapLogIdx[inputHash.(common.Hash).Hex()])
					constant.MapLogIdx["0x"+mcsTx] = constant.MapLogIdx[inputHash.(common.Hash).Hex()]
					w.log.Info("Success idx ", "des", "0x"+mcsTx, "idx", constant.MapLogIdx["0x"+mcsTx])
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
			errorCount++
			if errorCount >= 10 {
				w.mosAlarm(inputHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) sendTx(addr, method string, input []byte, txAmount, mul, used int64, ignore bool) (string, error) {
	// online estimateEnergy
	contract, err := w.conn.cli.TriggerConstantContractByEstimate(w.cfg.From, addr, input, txAmount)
	if err != nil {
		w.log.Error("Failed to TriggerConstantContract EstimateEnergy", "err", err)
		return "", err
	}

	for _, v := range contract.ConstantResult {
		w.log.Info("contract result", "err", string(v), "v", v, "hex", common.Bytes2Hex(v))
		ele := strings.TrimSpace(string(v))
		if ele != "" && ele != "4,^" && ele != "0�" && !ignore {
			return "", errors.New(ele)
		}
	}

	if used == 0 {
		used = contract.EnergyUsed
	}

	// // testnet
	// estimate, err := w.conn.cli.EstimateEnergy(w.cfg.From, addr, input, 0, "", 0)
	// if err != nil {
	// 	w.log.Error("Failed to EstimateEnergy", "err", err)
	// 	return "", err
	// }
	feeLimit := big.NewInt(0).Mul(big.NewInt(used), big.NewInt(420*mul))
	w.log.Info("EstimateEnergy", "estimate", used, "multiple", multiple, "feeLimit", feeLimit, "mul", mul)

	acc, err := w.conn.cli.GetAccountResource(w.cfg.From)
	if err != nil {
		return "", errors.Wrap(err, "get account failed")
	}
	// 22000 > (40000 - 10000) = false, 继续执行
	if used >= (acc.EnergyLimit-acc.EnergyUsed) && method != "rent" {
		//if estimate.EnergyRequired >= account.EnergyLimit {
		return "", fmt.Errorf("txUsed(%d) energy more than acount have(%d)", used, acc.EnergyLimit)
	}

	tx, err := w.conn.cli.TriggerContract(w.cfg.From, addr, input, feeLimit.Int64(), txAmount, "", 0)
	if err != nil {
		w.log.Error("Failed to TriggerContract", "err", err)
		return "", err
	}

	ks, acc, err := store.UnlockedKeystore(w.cfg.From, string(w.pass))
	if err != nil {
		w.log.Error("Failed to UnlockedKeystore", "err", err)
		return "", err
	}
	controller := transaction.NewController(w.conn.cli, ks, acc, tx.Transaction)
	if err = controller.ExecuteTransaction(); err != nil {
		w.log.Error("Failed to ExecuteTransaction", "err", err)
		return "", err
	}
	if controller.GetResultError() != nil {
		return "", fmt.Errorf("contro resultError is %v", controller.GetResultError())
	}
	return common.Bytes2Hex(tx.GetTxid()), nil
}

func (w *Writer) txStatus(txHash string) error {
	var count int64
	time.Sleep(time.Second * 2)
	for {
		id, err := w.conn.cli.GetTransactionInfoByID(txHash)
		if err != nil {
			w.log.Error("Failed to GetTransactionByID", "err", err)
			time.Sleep(constant.QueryRetryInterval)
			count++
			if count == 60 {
				return err
			}
			continue
		}
		if id.Receipt.Result == core.Transaction_Result_SUCCESS {
			w.log.Info("Tx receipt status is success", "hash", txHash)
			return nil
		}
		return fmt.Errorf("txHash(%s), status not success, current status is (%s)", txHash, id.Receipt.Result.String())
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2tron failed, srcHash=%v err is %s", tx, err.Error()))
}

func (w *Writer) checkOrderId(toAddress string, input []byte) (bool, error) {
	param := fmt.Sprintf("[{\"bytes32\":\"%v\"}]", common.Bytes2Hex(input))
	call, err := w.conn.cli.TriggerConstantContract(w.cfg.From, toAddress, "orderList(bytes32)", param)
	if err != nil {
		return false, fmt.Errorf("call orderList failed, %v", err.Error())
	}

	resp, err := mapprotocol.Mcs.Methods[mapprotocol.MethodOfOrderList].Outputs.Unpack(call.ConstantResult[0])
	if err != nil {
		return false, errors.Wrap(err, "output Unpack failed")
	}

	var exist bool
	err = mapprotocol.Mcs.Methods[mapprotocol.MethodOfOrderList].Outputs.Copy(&exist, resp)
	if err != nil {
		return false, errors.Wrap(err, "checkOrderId output copy failed")
	}
	return exist, nil
}

var (
	wei = big.NewFloat(1000000)
)

func (w *Writer) rentEnergy(used int64, method string) error {
	acc, err := w.conn.cli.GetAccountResource(w.cfg.From)
	if err != nil {
		return err
	}
	if w.cfg.FeeType == constant.FeeRentType {
		return w.feeRentEnergy(used, acc)
	}

	account, err := w.conn.cli.GetAccount(w.cfg.From)
	if err != nil {
		return err
	}
	balance, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(account.Balance), wei).Float64()
	w.log.Info("Rent energy, account energy detail", "account", w.cfg.From, "all", acc.EnergyLimit, "used", acc.EnergyUsed,
		"trx", balance)
	if method == mapprotocol.MtdOfSwapInVerifiedWithIndex || method == mapprotocol.MethodOfSwapInVerified {
		w.log.Info("Rent energy, call method is swapInVerified or withIndex, dont need rent energy", "method", method)
		return nil
	}
	mul := float64(used) * 1.1
	if (acc.EnergyLimit - acc.EnergyUsed) > int64(mul) {
		w.log.Info("Rent energy, account have enough energy", "account", w.cfg.From,
			"have", acc.EnergyLimit-acc.EnergyUsed, "estimate", mul)
		return nil
	}

	if balance < 400 {
		return errors.New("account not have enough balance(400 trx)")
	}

	input, err := mapprotocol.TronAbi.Pack("rentResource", w.cfg.EthFrom,
		big.NewInt(171558000000), big.NewInt(1))
	if err != nil {
		return errors.Wrap(err, "pack input failed")
	}
	w.log.Info("Rent energy will rent")
	tx, err := w.sendTx(w.cfg.RentNode, "rent", input, 371019500, 1, 0, false)
	if err != nil {
		return errors.Wrap(err, "sendTx failed")
	}
	w.log.Info("Rent energy success", "tx", tx)
	err = w.txStatus(tx)
	if err != nil {
		w.log.Warn("TxHash Status is not successful, will retry", "err", err)
		return err
	}

	return nil
}

func (w *Writer) feeRentEnergy(used int64, acc *api.AccountResourceMessage) error {
	resValue := int64(1000000)
	rentDuration := int64(1)
	if acc.EnergyLimit > used {
		w.log.Info("FeeRentEnergy dont need rent, because account have enough energy", "have", acc.EnergyLimit, "used", used)
		return nil
	}
	account, err := w.conn.cli.GetAccount(w.cfg.From)
	if err != nil {
		return err
	}
	balance, _ := big.NewFloat(0).Quo(big.NewFloat(0).SetInt64(account.Balance), wei).Float64()
	res, err := GetOrderPrice(w.cfg.FeeKey, resValue, rentDuration)
	if err != nil {
		return err
	}
	if res.Data.PayAmount > balance {
		return fmt.Errorf("account not have enough balance(%0.4f trx)", res.Data.PayAmount)
	}
	ret, err := OrderSubmit(w.cfg.FeeKey, w.cfg.From, resValue, rentDuration)
	if err != nil {
		return err
	}
	w.log.Info("FeeRentEnergy rent success", "no", ret.Data.OrderNo)
	return nil
}

func (w *Writer) newReturn(method string) {
	if w.cfg.FeeType == constant.FeeRentType || (method != mapprotocol.MtdOfSwapInVerifiedWithIndex && method != mapprotocol.MethodOfSwapInVerified) {
		w.log.Info("Return energy, call method is not swapInVerified or withIndex, dont need return energy", "method", method)
		return
	}
	w.log.Info("Return energy will start")
	time.Sleep(constant.BlockRetryInterval)
	acc, err := w.conn.cli.GetAccountResource(w.cfg.From)
	if err != nil {
		w.log.Error("Return energy, GetAccountResource failed", "err", err)
		return
	}
	if acc.EnergyLimit <= 0 {
		w.log.Info("Return energy, user not rent energy", "gas", acc.EnergyLimit)
		return
	}
	input, err := mapprotocol.TronAbi.Pack("returnResource", w.cfg.EthFrom, big.NewInt(171558000000), big.NewInt(1))
	if err != nil {
		w.log.Error("Return energy, Pack failed", "err", err)
		return
	}
	tx, err := w.sendTx(w.cfg.RentNode, "return", input, 0, 1, 80000, true)
	if err != nil {
		w.log.Error("Return energy, sendTx failed", "err", err)
		return
	}
	w.log.Info("Return energy success", "tx", tx)
}
