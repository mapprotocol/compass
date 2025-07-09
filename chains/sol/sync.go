package sol

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/abi"
	"github.com/mapprotocol/compass/internal/contract"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mr-tron/base58"
	"math/big"
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
	handler   Handler
	conn      core.Connection
	cfg       *Config
	solClient *rpc.Client
}

func newSync(cs *chain.CommonSync, handler Handler, conn core.Connection, cfg *Config, solClient *rpc.Client) *sync {
	return &sync{CommonSync: cs, handler: handler, conn: conn, cfg: cfg, solClient: solClient}
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

type CrossOutData struct {
	Relay                     bool   `json:"relay"`
	OrderId                   string `json:"orderId"`
	TokenAmount               string `json:"tokenAmount"`
	From                      []byte `json:"from"`
	FromToken                 []byte `json:"fromToken"`
	ToToken                   []byte `json:"toToken"`
	SwapTokenOut              string `json:"swapTokenOut"`
	SwapTokenOutMinAmountOut  string `json:"swapTokenOutMinAmountOut"`
	MinAmountOut              string `json:"minAmountOut"`
	SwapTokenOutBeforeBalance string `json:"swapTokenOutBeforeBalance"`
	AfterBalance              string `json:"afterBalance"`
	Receiver                  string `json:"receiver"`
	OriginReceiver            []byte `json:"originReceiver"`
	ToChain                   string `json:"toChain"`
	FromChainId               string `json:"fromChainId"`
	AmountOut                 string `json:"amountOut"`
	RefererId                 []int  `json:"refererId"`
	FeeRatio                  []int  `json:"feeRatio"`
	SwapData                  string `json:"swapData"`
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
			m.Cfg.StartBlock.Int64(), constant.ProjectOfOther, m.Cfg.Id, topic)))
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

func mosHandler(m *sync) (int64, error) {
	// 通过 filter 过滤
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil || log.Id == 0 {
		return 0, nil
	}
	receiptHash, receiptPack, err := m.genReceipt(log)
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

	tmpData := CrossOutData{}
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

	err = m.checkLog(log)
	if err != nil {
		return 0, err
	}

	receiptHash, _, err := m.genReceipt(log)
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

const (
	UsdcOfSol = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	WsolOfSol = "So11111111111111111111111111111111111111112"
)

func (m *sync) genReceipt(log *Log) (*common.Hash, []byte, error) {
	// 解析
	tmpData := CrossOutData{}
	err := json.Unmarshal([]byte(log.Data), &tmpData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal resp.Data failed")
	}
	fromChain, ok := big.NewInt(0).SetString(tmpData.FromChainId, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid fromChain (%s)", tmpData.FromChainId)
	}
	toChain, ok := big.NewInt(0).SetString(tmpData.ToChain, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid toChain (%s)", tmpData.ToChain)
	}
	amount, ok := big.NewInt(0).SetString(tmpData.AmountOut, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amount (%s)", tmpData.TokenAmount)
	}
	minAmount, ok := big.NewInt(0).SetString(tmpData.MinAmountOut, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid minAmount (%s)", tmpData.MinAmountOut)
	}
	orderId := common.HexToHash(tmpData.OrderId)
	token := tmpData.ToToken[:20]
	form := tmpData.From
	to := common.Hex2Bytes(strings.TrimPrefix(tmpData.Receiver, "0x"))

	bridgeParam := &abi.BridgeParam{}
	if len(tmpData.SwapData) > 0 {
		bridgeParam, err = abi.DecodeBridgeParam(common.Hex2Bytes(strings.TrimPrefix(tmpData.SwapData, "0x")))
		if err != nil {
			return nil, nil, errors.Wrap(err, "decode bridge param failed")
		}

		if len(bridgeParam.SwapData) > 0 {
			// check swapData
			rece := tmpData.OriginReceiver[13:]
			if toChain.Int64() == constant.BtcChainId {
				if isFirst12Zero(tmpData.OriginReceiver) {
					rece = tmpData.OriginReceiver[12:]
				} else {
					rece = tmpData.OriginReceiver[:]
				}
			}
			if common.Bytes2Hex(token) == "0000000000000000000000000000000000425443" {
				token = common.Hex2Bytes("425443")
			}
			pass, err := contract.Validate(tmpData.Relay, toChain, minAmount, token, rece, bridgeParam.SwapData)
			if err != nil {
				return nil, nil, err
			}
			if !pass {
				return nil, nil, fmt.Errorf("invalid swapData (%s)", tmpData.SwapData)
			}
		}
	}

	bridgeToken := make([]byte, 0)
	if tmpData.SwapTokenOut == m.cfg.UsdcAda {
		bridgeToken, _ = base58.Decode(UsdcOfSol)
	} else if tmpData.SwapTokenOut == m.cfg.WsolAda {
		bridgeToken, _ = base58.Decode(WsolOfSol)
	}
	eo := mapprotocol.MessageOutEvent{
		FromChain:   fromChain,
		ToChain:     toChain,
		OrderId:     orderId,
		Amount:      amount,
		Token:       bridgeToken,
		From:        form,
		SwapData:    bridgeParam.SwapData,
		GasLimit:    big.NewInt(0),
		Mos:         common.Hex2Bytes("0000317bec33af037b5fab2028f52d14658f6a56"),
		Initiator:   form,
		Relay:       tmpData.Relay,
		MessageType: uint8(3),
		To:          to,
	}

	fmt.Println("eo.From", base58.Encode(eo.From))
	fmt.Println("eo.Initiator", base58.Encode(eo.Initiator))
	fmt.Println("eo.Relay", eo.Relay)
	fmt.Println("eo.to", common.Bytes2Hex(eo.To))
	fmt.Println("eo.FromChain", eo.FromChain)
	fmt.Println("eo.ToChain", eo.ToChain)
	fmt.Println("eo.OrderId", eo.OrderId)
	fmt.Println("eo.Amount", eo.Amount)
	fmt.Println("eo.Token", common.Bytes2Hex(eo.Token))
	fmt.Println("eo.SwapData", common.Bytes2Hex(bridgeParam.SwapData))

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
	receiptPack, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolPackReceipt].Inputs.Pack(addr, []byte("MessageOutEvent"), data) // temp
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal sol pack failed")
	}
	receipt := common.BytesToHash(crypto.Keccak256(receiptPack))
	return &receipt, receiptPack, nil
}

func isFirst12Zero(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	for i := 0; i < 12; i++ {
		if data[i] != 0 {
			return false
		}
	}
	return true
}

func (m *sync) checkLog(target *Log) error {
	sig := solana.MustSignatureFromBase58(target.TxHash)
	txResult, err := m.solClient.GetTransaction(context.Background(), sig, &rpc.GetTransactionOpts{
		Commitment:                     rpc.CommitmentFinalized,
		MaxSupportedTransactionVersion: &rpc.MaxSupportedTransactionVersion0,
		Encoding:                       solana.EncodingBase64,
	})
	if err != nil {
		return err
	}
	tx, err := txResult.Transaction.GetTransaction()
	if err != nil {
		return err
	}
	if len(tx.Message.Instructions) == 0 {
		return nil
	}

	tmpData := CrossOutData{}
	err = json.Unmarshal([]byte(target.Data), &tmpData)
	if err != nil {
		return errors.Wrap(err, "unmarshal resp.Data failed")
	}

	ev := &CrossFinishEvent{}
	eventPrefix := "Program data: "
	for _, msg := range txResult.Meta.LogMessages {
		if !strings.HasPrefix(msg, eventPrefix) {
			continue
		}
		base64Data := strings.TrimPrefix(msg, eventPrefix)
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			fmt.Println("base64 decode failed", err)
			continue
		}

		ev, err = parseCrossFinishEventData(data)
		if err != nil {
			continue
		}
	}
	if ev == nil {
		return fmt.Errorf("invalid CrossFinishEvent, hash(%s)", target.TxHash)
	}

	ab, _ := big.NewInt(0).SetString(tmpData.AfterBalance, 16)
	if tmpData.OrderId != common.Bytes2Hex(ev.OrderRecord.OrderId) && ab.Uint64() == ev.AfterBalance {
		return errors.New("tx log not match")
	}

	return nil
}
