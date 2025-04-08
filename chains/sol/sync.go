package sol

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

type Handler func(*sync) (int64, error)

type sync struct {
	*chain.CommonSync
	handler Handler
	conn    core.Connection
	cfg     *Config
}

func newSync(cs *chain.CommonSync, handler Handler, conn core.Connection, cfg *Config) *sync {
	return &sync{CommonSync: cs, handler: handler, conn: conn, cfg: cfg}
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
				util.Alarm(context.Background(), fmt.Sprintf("filter mos failed, chain=%s, err is %s", m.Cfg.Name, err.Error()))
				time.Sleep(constant.BlockRetryInterval)
				continue
			}
			if id == 0 {
				time.Sleep(constant.MessengerInterval)
				continue
			}

			m.Cfg.StartBlock = big.NewInt(id)
			_ = m.WaitUntilMsgHandled(1)
			err = m.BlockStore.StoreBlock(m.Cfg.StartBlock)
			if err != nil {
				m.Log.Error("Failed to write latest block to blockstore", "err", err)
			}

			time.Sleep(constant.MessengerInterval)
		}
	}
}

type Log struct {
	Id          int64  `json:"id"`
	BlockNumber int64  `json:"blockNumber"`
	Addr        string `json:"addr"`
	Topic       string `json:"topic"`
	Data        string `json:"data"`
	TxHash      string `json:"txHash"`
}

type LogData struct {
	OrderId     string `json:"orderId"`
	Relay       int    `json:"relay"`
	MessageType int    `json:"messageType"`
	FromChain   string `json:"fromChain"`
	ToChain     string `json:"toChain"`
	Mos         string `json:"mos"`
	Token       string `json:"token"`
	Initiator   string `json:"initiator"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	GasLimit    string `json:"gasLimit"`
	SwapData    string `json:"swapData"`
}

func filter(m *sync) (*Log, error) {
	topic := ""
	for idx, ele := range m.cfg.SolEvent {
		topic += ele
		if idx != len(m.cfg.SolEvent)-1 {
			topic += ","
		}
	}

	data, err := chain.Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), constant.ProjectOfMsger, m.Cfg.Id, topic)))
	if err != nil {
		return nil, err
	}
	listData, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "marshal resp.Data failed")
	}
	back := stream.MosListResp{}
	err = json.Unmarshal(listData, &back)
	if err != nil {
		return nil, err
	}
	if len(back.List) == 0 {
		return nil, nil
	}

	var ret = Log{}
	for _, ele := range back.List {
		idx := match(ele.ContractAddress, m.cfg.McsContract)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress, "txHash", ele.TxHash)
			m.Cfg.StartBlock = big.NewInt(ele.Id)
			continue
		}

		split := strings.Split(ele.Topic, ",")
		topics := make([]common.Hash, 0, len(split))
		for _, sp := range split {
			topics = append(topics, common.HexToHash(sp))
		}
		ret = Log{
			Id:          ele.Id,
			BlockNumber: int64(ele.BlockNumber),
			Addr:        ele.ContractAddress,
			Topic:       ele.Topic,
			Data:        ele.LogData,
			TxHash:      ele.TxHash,
		}

		m.Log.Info("Filter find Log", "id", ele.Id, "txHash", ele.TxHash)
	}

	return &ret, nil
}

func match(addr string, target []string) int {
	for idx, ele := range target {
		if ele == addr {
			return idx
		}
	}
	return -1
}

func messagerHandler(m *sync) (int64, error) {
	// 通过 filter 过滤
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil || log.Id == 0 {
		return 0, nil
	}
	receiptHash, receiptPack, err := genReceipt(log)
	if err != nil {
		return 0, errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Sol2Evm msger generate", "receiptHash", receiptHash)
	proposalInfo, err := chain.GetSigner(big.NewInt(log.BlockNumber), *receiptHash, uint64(m.cfg.Id), uint64(m.cfg.MapChainID))
	if err != nil {
		return 0, err
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
		return 0, errors.Wrap(err, "pack getBytes failed")
	}

	tmpData := LogData{}
	err = json.Unmarshal([]byte(log.Data), &tmpData)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshal resp.Data failed")
	}
	orderId := common.HexToHash(tmpData.OrderId)
	finalInput, err := mapprotocol.PackInput(mapprotocol.Mcs, mapprotocol.MethodOfMessageIn,
		big.NewInt(0).SetUint64(uint64(m.Cfg.Id)),
		big.NewInt(int64(0)), orderId, input)
	if err != nil {
		return 0, nil
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
		return 0, nil
	}

	return log.Id, nil
}

func oracleHandler(m *sync) (int64, error) {
	// 通过 filter 过滤
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil || log.Id == 0 {
		return 0, nil
	}

	receiptHash, _, err := genReceipt(log)
	if err != nil {
		return 0, errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Sol2Evm oracle generate", "receiptHash", receiptHash)

	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.MapChainID))
	if err != nil {
		return 0, errors.Wrap(err, "mul sign failed")
	}

	version := make([]byte, 0)
	for _, v := range ret.Version {
		version = append(version, byte(v))
	}

	bn := big.NewInt(log.BlockNumber)
	input, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receiptHash,
		ret.Version, bn, big.NewInt(int64(m.Cfg.Id)))
	if err != nil {
		return 0, errors.Wrap(err, "oracle pack input failed")
	}

	message := msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, receiptHash, bn}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return 0, nil
	}

	return log.Id, nil
}

func genReceipt(log *Log) (*common.Hash, []byte, error) {
	// 解析
	tmpData := LogData{}
	err := json.Unmarshal([]byte(log.Data), &tmpData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal resp.Data failed")
	}
	fromChain, ok := big.NewInt(0).SetString(tmpData.FromChain, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid fromChain (%s)", tmpData.FromChain)
	}
	toChain, ok := big.NewInt(0).SetString(tmpData.ToChain, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid toChain (%s)", tmpData.ToChain)
	}
	amount, ok := big.NewInt(0).SetString(tmpData.Amount, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amount (%s)", tmpData.Amount)
	}
	gasLimit, ok := big.NewInt(0).SetString(tmpData.GasLimit, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid gasLimit (%s)", tmpData.GasLimit)
	}

	orderId := common.HexToHash(tmpData.OrderId)
	token := common.Hex2Bytes(tmpData.Token)
	form := common.Hex2Bytes(tmpData.From)
	to := common.Hex2Bytes(tmpData.To)
	initiator := common.Hex2Bytes(tmpData.Initiator)
	swapData := common.Hex2Bytes(tmpData.SwapData)
	mos := common.Hex2Bytes(tmpData.Mos)
	relayBool, err := strconv.ParseBool(fmt.Sprintf("%x", tmpData.Relay))
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse relay flag failed")
	}

	eo := mapprotocol.MessageOutEvent{
		FromChain:   fromChain,
		ToChain:     toChain,
		OrderId:     orderId,
		Amount:      amount,
		Token:       token,
		From:        form,
		SwapData:    swapData,
		GasLimit:    gasLimit,
		Mos:         mos,
		Initiator:   initiator,
		Relay:       relayBool,
		MessageType: uint8(tmpData.MessageType),
		To:          to,
	}
	data, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Pack(&eo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal event code failed")
	}
	fmt.Println("log.Addr ", log.Addr)
	addr := make([]byte, 0, 64)
	for _, ele := range solana.MustPublicKeyFromBase58(log.Addr) {
		addr = append(addr, ele)
	}
	// abi
	receiptPack, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolPackReceipt].Inputs.Pack(addr, []byte(log.Topic), data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal sol pack failed")
	}
	receipt := common.BytesToHash(crypto.Keccak256(receiptPack))
	return &receipt, receiptPack, nil
}
