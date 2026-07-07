package sol

import (
	"context"

	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/abi"
	"github.com/mapprotocol/compass/internal/contract"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mr-tron/base58"

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
	BridgeMint                string `json:"bridgeMint"`
	BridgeAmount              string `json:"bridgeAmount"`
}

var (
	anchorEventCpiDiscriminator = []byte{0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d}
	solAnchorEventNames         = map[string]string{
		string(anchorDiscriminator("event:CrossOutEvent")):       "CrossOutEvent",
		string(anchorDiscriminator("event:CrossInEvent")):        "CrossInEvent",
		string(anchorDiscriminator("event:ReleaseEvent")):        "ReleaseEvent",
		string(anchorDiscriminator("event:RefundEvent")):         "RefundEvent",
		string(anchorDiscriminator("event:MinOutOverrideEvent")): "MinOutOverrideEvent",
		string(anchorDiscriminator("event:CrossFinishEvent")):    "CrossFinishEvent",
	}
)

type solAnchorEvent struct {
	Name      string
	OrderId   []byte
	AmountOut *uint64
}

func filter(m *sync) (*Log, error) {
	topic := chain.BuildRawFilterTopic(m.cfg.SolEvent)
	back, err := m.ListMosLogs(constant.ProjectOfOther, topic, 1)
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
	bn := proof.GenLogBlockNumber(big.NewInt(log.BlockNumber), 1, uint(log.Id-4249250))
	proposalInfo, err := chain.GetSigner(bn, *receiptHash, uint64(m.cfg.Id), uint64(m.cfg.MapChainID))
	if err != nil {
		return 0, err
	}
	var fixedHash [32]byte
	for i, v := range receiptHash {
		fixedHash[i] = v
	}
	pd := proof.SignLogData{
		ProofType:   1,
		BlockNum:    bn,
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

	message := msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{finalInput,
		orderId, bn, log.TxHash}, m.MsgCh)
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
	bn := proof.GenLogBlockNumber(big.NewInt(log.BlockNumber), 1, uint(log.Id-4249250))

	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.MapChainID))
	if err != nil {
		return 0, errors.Wrap(err, "mul sign failed")
	}

	version := make([]byte, 0)
	for _, v := range ret.Version {
		version = append(version, byte(v))
	}

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
	amount, ok := big.NewInt(0).SetString(tmpData.BridgeAmount, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amount (%s)", tmpData.BridgeAmount)
	}
	minAmount, ok := big.NewInt(0).SetString(tmpData.MinAmountOut, 16)
	if !ok {
		return nil, nil, fmt.Errorf("invalid minAmount (%s)", tmpData.MinAmountOut)
	}
	orderId := common.HexToHash(tmpData.OrderId)
	token := tmpData.ToToken[12:]
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

	bridgeToken, _ := base58.Decode(tmpData.BridgeMint)
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
	if txResult.Meta == nil {
		return fmt.Errorf("missing transaction meta, hash(%s)", target.TxHash)
	}
	tx, err := txResult.Transaction.GetTransaction()
	if err != nil {
		return err
	}
	if len(tx.Message.Instructions) == 0 && len(txResult.Meta.InnerInstructions) == 0 {
		return nil
	}

	tmpData := CrossOutData{}
	err = json.Unmarshal([]byte(target.Data), &tmpData)
	if err != nil {
		return errors.Wrap(err, "unmarshal resp.Data failed")
	}

	targetOrderId := common.HexToHash(tmpData.OrderId).Bytes()
	ev := m.findSolAnchorEvent(txResult, tx, target.Addr, targetOrderId)
	if ev == nil {
		return fmt.Errorf("invalid sol anchor event, hash(%s), orderId(%s)", target.TxHash, tmpData.OrderId)
	}

	ab, _ := big.NewInt(0).SetString(tmpData.AmountOut, 16)
	if !bytes.Equal(targetOrderId, ev.OrderId) {
		return errors.New("tx log not match")
	}
	if ev.AmountOut != nil && ab.Uint64() != *ev.AmountOut {
		return errors.New("tx log amount not match")
	}

	return nil
}

func (m *sync) findSolAnchorEvent(txResult *rpc.GetTransactionResult, tx *solana.Transaction, logAddr string, targetOrderId []byte) *solAnchorEvent {
	if txResult == nil || txResult.Meta == nil {
		return nil
	}
	if ev := findSolAnchorEventFromLogs(txResult.Meta.LogMessages, targetOrderId); ev != nil {
		return ev
	}
	return m.findSolAnchorEventFromInnerInstructions(txResult.Meta.InnerInstructions, tx, txResult.Meta.LoadedAddresses, logAddr, targetOrderId)
}

func findSolAnchorEventFromLogs(logMessages []string, targetOrderId []byte) *solAnchorEvent {
	const eventPrefix = "Program data: "
	for _, msg := range logMessages {
		if !strings.HasPrefix(msg, eventPrefix) {
			continue
		}
		base64Data := strings.TrimPrefix(msg, eventPrefix)
		data, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			fmt.Println("base64 decode failed", err)
			continue
		}
		ev, err := parseCrossFinishEventData(data)
		if err == nil && bytes.Equal(targetOrderId, ev.OrderRecord.OrderId) {
			return &solAnchorEvent{
				Name:      "CrossFinishEvent",
				OrderId:   ev.OrderRecord.OrderId,
				AmountOut: &ev.AmountOut,
			}
		}
	}
	return nil
}

func (m *sync) findSolAnchorEventFromInnerInstructions(innerInstructions []rpc.InnerInstruction, tx *solana.Transaction, loadedAddresses rpc.LoadedAddresses, logAddr string, targetOrderId []byte) *solAnchorEvent {
	accountKeys := solanaLoadedAccountKeys(tx, loadedAddresses)
	programFilter := m.solEventProgramFilter(logAddr)
	for _, inner := range innerInstructions {
		for _, inst := range inner.Instructions {
			if int(inst.ProgramIDIndex) >= len(accountKeys) {
				continue
			}
			programID := accountKeys[inst.ProgramIDIndex].String()
			if len(programFilter) != 0 && !programFilter[programID] {
				continue
			}
			data := []byte(inst.Data)
			if len(data) <= len(anchorEventCpiDiscriminator) {
				continue
			}
			if !bytes.Equal(data[:len(anchorEventCpiDiscriminator)], anchorEventCpiDiscriminator) {
				continue
			}
			ev := parseSolAnchorEventCpi(data[len(anchorEventCpiDiscriminator):], targetOrderId)
			if ev != nil {
				return ev
			}
		}
	}
	return nil
}

func parseSolAnchorEventCpi(data []byte, targetOrderId []byte) *solAnchorEvent {
	const eventDiscriminatorLen = 8
	if len(data) < eventDiscriminatorLen+len(targetOrderId) {
		return nil
	}
	name, ok := solAnchorEventNames[string(data[:eventDiscriminatorLen])]
	if !ok {
		return nil
	}
	body := data[eventDiscriminatorLen:]
	idx := bytes.Index(body, targetOrderId)
	if idx == -1 {
		return nil
	}
	orderId := make([]byte, len(targetOrderId))
	copy(orderId, targetOrderId)
	return &solAnchorEvent{
		Name:    name,
		OrderId: orderId,
	}
}

func solanaLoadedAccountKeys(tx *solana.Transaction, loadedAddresses rpc.LoadedAddresses) solana.PublicKeySlice {
	if tx == nil {
		return nil
	}
	accountKeys := make(solana.PublicKeySlice, 0, len(tx.Message.AccountKeys)+len(loadedAddresses.Writable)+len(loadedAddresses.ReadOnly))
	accountKeys = append(accountKeys, tx.Message.AccountKeys...)
	accountKeys = append(accountKeys, loadedAddresses.Writable...)
	accountKeys = append(accountKeys, loadedAddresses.ReadOnly...)
	return accountKeys
}

func (m *sync) solEventProgramFilter(logAddr string) map[string]bool {
	programs := make(map[string]bool)
	for _, program := range m.cfg.McsContract {
		if program != "" {
			programs[program] = true
		}
	}
	if m.cfg.MessageIn != "" {
		programs[m.cfg.MessageIn] = true
	}
	if m.cfg.LightNode != "" {
		programs[m.cfg.LightNode] = true
	}
	if logAddr != "" {
		programs[logAddr] = true
	}
	return programs
}

func anchorDiscriminator(name string) []byte {
	hash := sha256.Sum256([]byte(name))
	return hash[:8]
}
