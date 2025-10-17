package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/report"
	"github.com/mapprotocol/compass/pkg/msg"

	"github.com/lbtsm/gotron-sdk/pkg/proto/api"

	"github.com/lbtsm/gotron-sdk/pkg/store"

	"github.com/pkg/errors"

	"github.com/lbtsm/gotron-sdk/pkg/proto/core"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lbtsm/gotron-sdk/pkg/client/transaction"
	"github.com/lbtsm/gotron-sdk/pkg/keystore"
	"github.com/mapprotocol/compass/internal/constant"
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
	isRent bool
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
	case msg.SwapWithMapProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var errorCount, checkIdCount int64
	addr := w.cfg.McsContract[m.Idx]
	orderId32 := m.Payload[1].(common.Hash)
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
				time.Sleep(time.Second * 10)
				continue
			}
			for _, v := range contract.ConstantResult {
				w.log.Info("Contract result", "err", string(v), "v", v)
				ele := strings.TrimSpace(string(v))
				internalErr := "0x" + hex.EncodeToString(v)
				if ele == "" {
					continue
				}
				for e := range constant.IgnoreError {
					if strings.Index(internalErr, e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "inputHash", inputHash, "err", internalErr)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				err = errors.New(internalErr)
			}
			if err != nil {
				w.mosAlarm(inputHash, fmt.Errorf("contract result failed, err is %v", err))
				time.Sleep(time.Second * 10)
				continue
			}
			w.log.Info("Trigger Contract result detail", "used", contract.EnergyUsed, "method", method)

			err = w.rentEnergy(contract.EnergyUsed, method)
			if err != nil {
				w.log.Info("Check energy failed", "srcHash", inputHash, "err", err)
				w.mosAlarm(inputHash, errors.Wrap(err, "please admin handler"))
				time.Sleep(time.Second * 10)
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
				} else {
					w.newReturn(method)
					report.Add(&report.Data{
						Hash:    mcsTx,
						IsRelay: false,
						OrderId: orderId32.Hex(),
					})
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
			w.newReturn(method)
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

	acco, err := w.conn.cli.GetAccountResource(w.cfg.From)
	if err != nil {
		return "", errors.Wrap(err, "get account failed")
	}
	final := float64(used) * 1.1
	// 22000 > (40000 - 10000) = false, continue exec
	if int64(final) >= (acco.EnergyLimit-acco.EnergyUsed) && method != "rent" && w.cfg.Rent {
		w.log.Info("SendTx EstimateEnergy", "err", "txUsed(%d) energy more than acount have(%d)", int64(final), acco.EnergyLimit)
		//if estimate.EnergyRequired >= account.EnergyLimit {
		return "", fmt.Errorf("txUsed(%d) energy more than acount have(%d)", int64(final), acco.EnergyLimit)
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

	if len(call.ConstantResult) == 0 {
		return false, fmt.Errorf("call orderList result empty")
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
	if !w.cfg.Rent {
		w.log.Info("dont need rent energy, cfg is false")
		return nil
	}
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

	mul := float64(used) * 1.1
	if (acc.EnergyLimit - acc.EnergyUsed) > int64(mul) {
		w.log.Info("Rent energy, account have enough energy", "account", w.cfg.From,
			"have", acc.EnergyLimit-acc.EnergyUsed, "estimate", mul)
		//return nil
	}

	if balance < 330 {
		return errors.New("account not have enough balance(330 trx)")
	}

	input, err := mapprotocol.TronAbi.Pack("rentResource", w.cfg.EthFrom,
		big.NewInt(122205000000), big.NewInt(1))
	if err != nil {
		return errors.Wrap(err, "pack input failed")
	}
	w.log.Info("Rent energy will rent")
	tx, err := w.sendTx(w.cfg.RentNode, "rent", input, 300000000, 1, 70000, false)
	if err != nil {
		return errors.Wrap(err, "sendTx failed")
	}
	w.log.Info("Rent energy success", "tx", tx)
	err = w.txStatus(tx)
	if err != nil {
		w.log.Warn("Rent TxHash Status is not successful, will retry", "err", err)
		return err
	}

	w.isRent = true
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
	if !w.isRent {
		w.log.Info("Return energy, is not rent, dont return")
		return
	}
	w.log.Info("Return energy will start")
	time.Sleep(constant.BlockRetryInterval)
	acc, err := w.conn.cli.GetAccountResource(w.cfg.From)
	if err != nil {
		w.log.Error("Return energy, GetAccountResource failed", "err", err)
		return
	}
	if acc.EnergyLimit-20004 <= 0 {
		w.log.Info("Return energy, user not rent energy", "gas", acc.EnergyLimit)
		return
	}
	input, err := mapprotocol.TronAbi.Pack("returnResource", w.cfg.EthFrom, big.NewInt(122205000000), big.NewInt(1))
	if err != nil {
		w.log.Error("Return energy, Pack failed", "err", err)
		return
	}
	tx, err := w.sendTx(w.cfg.RentNode, "return", input, 0, 1, 80000, true)
	if err != nil {
		w.log.Error("Return energy, sendTx failed", "err", err)
		return
	}
	err = w.txStatus(tx)
	if err != nil {
		w.log.Warn("Return TxHash Status is not successful, will retry", "err", err)
	}
	w.log.Info("Return energy success", "tx", tx)
	w.isRent = false
}
