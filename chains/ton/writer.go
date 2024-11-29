package ton

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	time.Sleep(24 * time.Hour)
	// todo ton 发交易

	packed := m.Payload[0].([]byte)
	receiptHash := m.Payload[1].(common.Hash)
	versionHash := m.Payload[2].(common.Hash)
	blockNumber := m.Payload[3].(int64)
	addr := m.Payload[4].(common.Address)
	topics := m.Payload[5].([]common.Hash)
	messages := m.Payload[6].([]byte)

	hash := crypto.Keccak256(packed)
	sign, err := chain.PersonalSign(string(hash), w.conn.Keypair().PrivateKey)
	if err != nil {
		w.log.Error("failed to sign", "error", err.Error())
		return false
	}
	sign[64] -= 27 // todo ton 签名需要减去27, 等合约中支持后删除
	signs := []*Signature{
		{
			V: new(big.Int).SetBytes([]byte{sign[64]}).Uint64(),
			R: new(big.Int).SetBytes(sign[0:32]),
			S: new(big.Int).SetBytes(sign[32:64]),
		},
	}

	marshal, _ := json.Marshal(signs)
	if err != nil {
		return false
	}
	fmt.Println("============================== to ton message in params: ", common.Bytes2Hex(hash), common.Bytes2Hex(sign), receiptHash, versionHash, blockNumber, int64(m.Source), addr, topics, common.Bytes2Hex(messages))
	fmt.Println("============================== hash: ", common.Bytes2Hex(hash))
	fmt.Println("============================== signs: ", string(marshal))
	fmt.Println("============================== receiptRoot: ", receiptHash)
	fmt.Println("============================== version: ", versionHash)
	fmt.Println("============================== blockNum: ", blockNumber)
	fmt.Println("============================== chainId: ", int64(m.Source))
	fmt.Println("============================== addr: ", addr)
	fmt.Println("============================== topics: ", topics)
	fmt.Println("============================== messages: ", common.Bytes2Hex(messages))
	fmt.Println("============================== mcs address: ", w.cfg.McsContract)

	dstAddr, err := address.ParseAddr(w.cfg.McsContract[m.Idx])
	if err != nil {
		w.log.Error("failed to parse ton address", "address", w.cfg.McsContract[m.Idx], "error", err.Error())
		return false
	}

	var errorCount int64
	for {
		select {
		case <-w.stop:
			return false
		default:
			cell, err := GenerateMessageInCell(common.BytesToHash(hash), signs, receiptHash, versionHash, blockNumber, int64(m.Source), addr, topics, messages)
			if err != nil {
				w.log.Error("failed to generate message in cell", "error", err.Error())
				continue
			}
			data, _ := cell.MarshalJSON()
			fmt.Println("============================== cell: ", common.Bytes2Hex(data))

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
			fmt.Println("============================== sent transaction to ton, error: ", err)
			if err == nil {
				w.log.Info("successful send transaction to ton", "src", m.Source, "dst", m.Destination, "txHash", hex.EncodeToString(tx.Hash))
				return true
			} else {
				data, _ := cell.MarshalJSON()
				w.log.Error("failed to send transaction to ton", "dstAddr", dstAddr, "body", string(data), "error", err)
			}

			errorCount++
			if errorCount >= 10 {
				// todo alarm
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
		}
	}
}
