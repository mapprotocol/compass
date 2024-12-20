package ton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/util"
	"math/big"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/lbtsm/gotron-sdk/pkg/keystore"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
)

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
	case msg.SwapWithMapProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

func (w *Writer) exeMcs(m msg.Message) bool {

	var (
		externalError error
		errorCount    int64
	)
	for {
		select {
		case <-w.stop:
			return false
		default:
			log := m.Payload[0].(*types.Log)

			if errorCount >= 10 {
				w.mosAlarm(log.TxHash, externalError)
				errorCount = 0
			}

			ret, err := chain.MulSignInfo(0, uint64(w.cfg.MapChainID))
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to get mul sign info", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			receiptHash, err := chain.GenLogReceipt(log)
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to gen log receipt", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			packed, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receiptHash, ret.Version, new(big.Int).SetUint64(log.BlockNumber), big.NewInt(int64(w.cfg.MapChainID)))
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to pack soliditypack input", "error", err)
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			hash := crypto.Keccak256(packed)
			sign, err := chain.PersonalSign(string(hash), w.conn.Keypair().PrivateKey)
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to personal sign", "error", err.Error())
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			signs := []*Signature{
				{
					V: new(big.Int).SetBytes([]byte{sign[64]}).Uint64(),
					R: new(big.Int).SetBytes(sign[0:32]),
					S: new(big.Int).SetBytes(sign[32:64]),
				},
			}

			dstAddr, err := address.ParseAddr(w.cfg.McsContract[m.Idx])
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to parse ton address", "address", w.cfg.McsContract[m.Idx], "error", err.Error())
				time.Sleep(constant.TxRetryInterval)
				continue
			}
			// todo remove debug log
			marshal, _ := json.Marshal(signs)
			fmt.Println("hash: ", common.Bytes2Hex(hash))
			fmt.Println("expectedAddress: ", w.conn.Keypair().Address)
			fmt.Println("signs: ", string(marshal))
			fmt.Println("receiptRoot: ", *receiptHash)
			fmt.Println("version: ", common.BytesToHash(ret.Version[:]))
			fmt.Println("blockNum: ", log.BlockNumber)
			fmt.Println("chainId: ", int64(m.Source))
			fmt.Println("addr: ", log.Address)
			fmt.Println("topics: ", log.Topics)
			fmt.Println("message: ", common.Bytes2Hex(log.Data))
			fmt.Println("mcs address: ", w.cfg.McsContract)

			cell, err := GenerateMessageInCell(common.BytesToHash(hash), w.conn.Keypair().Address, signs, *receiptHash, common.BytesToHash(ret.Version[:]), int64(log.BlockNumber), int64(m.Source), log.Address, log.Topics, log.Data)
			if err != nil {
				errorCount++
				externalError = err
				w.log.Error("failed to generate message in cell", "error", err.Error())
				time.Sleep(constant.TxRetryInterval)
				continue
			}

			// todo remove debug log
			data, _ := cell.MarshalJSON()
			fmt.Println("generated message in cell: ", string(data))

			message := &wallet.Message{
				Mode: wallet.PayGasSeparately, // pay fees separately (from balance, not from amount)
				InternalMessage: &tlb.InternalMessage{
					Bounce:  true, // return amount in case of processing error
					DstAddr: dstAddr,
					Amount:  tlb.MustFromTON("0.1"),
					Body:    cell,
				},
			}
			//err = w.conn.wallet.Send(context.Background(), message, false)
			tx, _, err := w.conn.wallet.SendWaitTransaction(context.Background(), message)
			if err == nil {
				w.log.Info("successful send transaction to ton", "src", m.Source, "srcHash", log.TxHash, "dst", m.Destination, "txHash", hex.EncodeToString(tx.Hash))
				m.DoneCh <- struct{}{}
				return true
			} else {
				errorCount++
				externalError = err
				data, _ := cell.MarshalJSON()
				w.log.Error("failed to send transaction to ton", "src", m.Source, "srcHash", log.TxHash, "dst", m.Destination, "dstAddr", dstAddr, "body", string(data), "error", err)
			}

			//errorCount++
			//if errorCount >= 10 {
			//	w.mosAlarm(log.TxHash, err)
			//	errorCount = 0
			//}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2ton failed, srcHash=%v err is %s", tx, err.Error()))
}
