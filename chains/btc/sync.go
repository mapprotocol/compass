package btc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

type (
	Handler    func(*sync) (int64, error)
	LogHandler func(*sync, *MessageOut) error
)

type sync struct {
	*chain.CommonSync
	handler Handler
	cfg     *Config
}

func newSync(cs *chain.CommonSync, handler Handler, cfg *Config) *sync {
	return &sync{CommonSync: cs, handler: handler, cfg: cfg}
}

func (m *sync) Sync() error {
	m.Log.Info("Starting listener...")
	if !m.Cfg.SyncToMap {
		time.Sleep(time.Hour * 2400)
		return nil
	}

	select {
	case <-m.Stop:
		return errors.New("polling terminated")
	default:
		for {
			id, err := m.handler(m)
			if err != nil {
				if errors.Is(err, chain.NotVerifyAble) {
					time.Sleep(constant.BlockRetryInterval)
					continue
				}
				m.Log.Error("Filter Failed to get events for block", "err", err)
				util.Alarm(context.Background(), fmt.Sprintf("handler mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if id == 0 {
				time.Sleep(constant.MessengerInterval)
				continue
			}

			_ = m.WaitUntilMsgHandled(1)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to file", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
}

func handler(lh LogHandler) Handler {
	return func(m *sync) (int64, error) {
		topic := ""
		for idx, ele := range m.cfg.Events {
			topic += ele.GetTopic().Hex()
			if idx != len(m.cfg.Events)-1 {
				topic += ","
			}
		}

		data, err := chain.Request(fmt.Sprintf("%s/%s?%s", m.Cfg.BtcHost, constant.FilterBtcLogUrl,
			fmt.Sprintf("id=%d&chain_id=%d&topic=%s&limit=1",
				m.Cfg.StartBlock.Int64(), m.Cfg.Id, topic)))
		if err != nil {
			return 0, err
		}
		listData, err := json.Marshal(data)
		if err != nil {
			return 0, errors.Wrap(err, "marshal resp.Data failed")
		}
		back := stream.BtcLogListResp{}
		err = json.Unmarshal(listData, &back)
		if err != nil {
			return 0, err
		}
		if len(back.Items) == 0 {
			return 0, nil
		}
		logData := common.Hex2Bytes(back.Items[0].LogData)
		var log = MessageOut{}
		err = json.Unmarshal(logData, &log)
		if err != nil {
			return 0, err
		}
		fmt.Println("logData ----------------- ", string(logData))
		log.Id = back.Items[0].Id
		log.Addr = m.cfg.Addr
		log.Topic = back.Items[0].Topic
		log.TxHash = back.Items[0].TxHash
		log.BlockNumber = back.Items[0].BlockNumber
		err = lh(m, &log)
		if err != nil {
			return 0, err
		}

		return 0, nil
	}
}

func mos(m *sync, log *MessageOut) error {
	receiptHash, receiptPack, err := genReceipt(log)
	if err != nil {
		return errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Sol2Evm mos generate", "receiptHash", receiptHash, "1111", "111")
	proposalInfo, err := getSigner(log.BlockNumber, *receiptHash, uint64(m.cfg.Id), uint64(m.cfg.MapChainID))
	if err != nil {
		return err
	}
	var fixedHash [32]byte
	for i, v := range receiptHash {
		fixedHash[i] = v
	}
	pd := proof.SignLogData{
		ProofType:   1,
		BlockNum:    big.NewInt(log.BlockNumber),
		ReceiptRoot: fixedHash,
		Signatures:  proposalInfo.Signatures,
		Proof:       receiptPack,
	}

	input, err := mapprotocol.GetAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return errors.Wrap(err, "pack getBytes failed")
	}

	orderId := common.HexToHash(log.OrderID)
	finalInput, err := mapprotocol.PackInput(mapprotocol.Mcs, mapprotocol.MethodOfMessageIn,
		big.NewInt(0).SetUint64(uint64(m.Cfg.Id)),
		big.NewInt(int64(0)), orderId, input)
	if err != nil {
		return err
	}

	var orderId32 [32]byte
	for i, v := range orderId {
		orderId32[i] = v
	}
	message := msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{finalInput,
		orderId32, log.BlockNumber, log.TxHash}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return err
	}

	return nil
}

func oracle(m *sync, log *MessageOut) error {
	receiptHash, _, err := genReceipt(log)
	if err != nil {
		return errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Sol2Evm oracle generate", "receiptHash", receiptHash)

	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.MapChainID))
	if err != nil {
		return errors.Wrap(err, "mul sign failed")
	}

	version := make([]byte, 0)
	for _, v := range ret.Version {
		version = append(version, byte(v))
	}

	bn := big.NewInt(log.BlockNumber)
	input, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receiptHash,
		ret.Version, bn, big.NewInt(int64(m.Cfg.Id)))
	if err != nil {
		return errors.Wrap(err, "oracle pack input failed")
	}

	message := msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, receiptHash, bn}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return nil
	}

	return nil
}

func genReceipt(log *MessageOut) (*common.Hash, []byte, error) {
	if len(log.SwapData) > 0 {
		// todo check swapData
	}
	// 解析
	fromChain, ok := big.NewInt(0).SetString(log.SrcChain, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid fromChain (%s)", log.SrcChain)
	}
	toChain, ok := big.NewInt(0).SetString(log.DstChain, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid toChain (%s)", log.DstChain)
	}
	amount, ok := big.NewInt(0).SetString(log.InAmount, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amount (%s)", log.InAmount)
	}
	gasLimit, ok := big.NewInt(0).SetString(log.GasLimit, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid gasLimit (%s)", log.GasLimit)
	}

	orderId := common.HexToHash(log.OrderID)

	swapData := common.Hex2Bytes(strings.TrimPrefix(log.SwapData, "0x"))
	MOS := common.Hex2Bytes(strings.TrimPrefix(log.MOS, "0x"))

	eo := mapprotocol.MessageOutEvent{
		FromChain:   fromChain,
		ToChain:     toChain,
		OrderId:     orderId,
		Amount:      amount,
		Token:       common.Hex2Bytes(strings.TrimPrefix(log.SrcToken, "0x")),
		From:        []byte(log.From), // btc address
		SwapData:    swapData,
		GasLimit:    gasLimit,
		Mos:         MOS,
		Initiator:   []byte(log.Sender), // btc address
		Relay:       log.Relay,
		MessageType: log.MessageType,
		To:          common.Hex2Bytes(strings.TrimPrefix(log.Receiver, "0x")),
	}
	data, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Pack(&eo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal event code failed")
	}
	// abi
	receiptPack, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolPackReceipt].Inputs.Pack(
		common.Hex2Bytes(strings.TrimPrefix(log.Addr, "0x")),
		common.Hex2Bytes(strings.TrimPrefix(log.Topic, "0x")), data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal sol pack failed")
	}
	receipt := common.BytesToHash(crypto.Keccak256(receiptPack))
	return &receipt, receiptPack, nil
}

type MessageOut struct {
	Id           int64  `json:"id"`
	Topic        string `json:"topic"`
	BlockNumber  int64  `json:"block_number"`
	TxHash       string `json:"tx_hash"`
	Addr         string `json:"addr"`
	OrderID      string `json:"order_id"`  // orderId
	From         string `json:"from"`      // relay
	To           string `json:"to"`        //
	SrcChain     string `json:"src_chain"` // fromChain
	SrcToken     string `json:"src_token"` // token
	Sender       string `json:"sender"`    // initiator
	InAmount     string `json:"in_amount"` // amount
	InTxHash     string `json:"in_tx_hash"`
	BridgeFee    string `json:"bridge_fee"`
	DstChain     string `json:"dst_chain"` // toChain
	DstToken     string `json:"dst_token"` //
	Receiver     string `json:"receiver"`  //
	MOS          string `json:"mos"`       // map mos address
	Relay        bool   //   (from butter)
	MessageType  uint8  // default 3
	GasLimit     string // default 0
	MinOutAmount string `json:"min_out_amount"` //  minOutAmount
	SwapData     string `json:"swap_data"`      // (from butter)
}

type T struct {
	OrderId      string `json:"order_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	SrcChain     string `json:"src_chain"`
	SrcToken     string `json:"src_token"`
	Sender       string `json:"sender"`
	InAmount     string `json:"in_amount"`
	InTxHash     string `json:"in_tx_hash"`
	BridgeFee    string `json:"bridge_fee"`
	DstChain     string `json:"dst_chain"`
	DstToken     string `json:"dst_token"`
	Receiver     string `json:"receiver"`
	Mos          string `json:"mos"`
	Relay        bool   `json:"relay"`
	MessageType  int    `json:"message_type"`
	GasLimit     string `json:"gas_limit"`
	MinOutAmount string `json:"min_out_amount"`
	SwapData     string `json:"swap_data"`
}

func getSigner(blockNumber int64, receiptHash common.Hash, selfId, toChainID uint64) (*chain.ProposalInfoResp, error) {
	bn := big.NewInt(blockNumber)
	ret, err := chain.MulSignInfo(0, toChainID)
	if err != nil {
		return nil, err
	}

	piRet, err := chain.ProposalInfo(0, selfId, toChainID, bn, receiptHash, ret.Version)
	if err != nil {
		return nil, err
	}

	if !piRet.CanVerify {
		return nil, chain.NotVerifyAble
	}

	return piRet, nil
}
