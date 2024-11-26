package sol

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/mapprotocol/compass/internal/stream"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

type Handler func(*sync) (int, error)

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
			count, err := m.handler(m)
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

			_ = m.WaitUntilMsgHandled(count)
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

func filter(m *sync) (*Log, error) {
	topic := ""
	for idx, ele := range m.cfg.SolEvent {
		topic += ele
		if idx != len(m.cfg.SolEvent)-1 {
			topic += ","
		}
	}
	fmt.Println("url --------- ", fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), constant.ProjectOfMsger, m.Cfg.Id, topic)))
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

	ret := &Log{}
	for _, ele := range back.List {
		idx := match(ele.ContractAddress, m.cfg.McsContract)
		if idx == -1 {
			m.Log.Info("Filter Log Address Not Match", "id", ele.Id, "address", ele.ContractAddress)
			m.Cfg.StartBlock = big.NewInt(ele.Id)
			continue
		}

		split := strings.Split(ele.Topic, ",")
		topics := make([]common.Hash, 0, len(split))
		for _, sp := range split {
			topics = append(topics, common.HexToHash(sp))
		}
		ret = &Log{
			Id:          ele.Id,
			BlockNumber: int64(ele.BlockNumber),
			Addr:        ele.ContractAddress,
			Topic:       ele.Topic,
			Data:        ele.LogData,
			TxHash:      ele.TxHash,
		}

		m.Log.Info("Filter find Log", "id", ele.Id, "txHash", ele.TxHash)
	}

	return ret, nil
}

func match(addr string, target []string) int {
	for idx, ele := range target {
		if ele == addr {
			return idx
		}
	}
	return -1
}

func messagerHandler(m *sync) (int, error) {
	// 通过 filter 过滤
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil {
		return 0, nil
	}
	// 解析
	tmp := make(map[string]string)
	err = json.Unmarshal([]byte(log.Data), &tmp)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshal resp.Data failed")
	}
	m.Log.Info("Sol2Evm msger parse success", "data", tmp)
	//routeOrder, err := DecodeRouteOrder(common.Hex2Bytes(tmp["payload"]))
	//if err != nil {
	//	return 0, errors.Wrap(err, "decode route order failed")
	//}

	return 0, nil
}

func oracleHandler(m *sync) (int, error) {
	// 通过 filter 过滤
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil {
		return 0, nil
	}
	// 解析
	data := make(map[string]string)
	err = json.Unmarshal([]byte(log.Data), &data)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshal resp.Data failed")
	}
	m.Log.Info("Sol2Evm oracle parse success", "data", data)
	routeOrder, err := DecodeRouteOrder(common.Hex2Bytes(data["payload"]))
	if err != nil {
		return 0, errors.Wrap(err, "decode route order failed")
	}
	receiptHash, err := genReceipt(m.Cfg.MapChainID, log, routeOrder, data)
	if err != nil {
		return 0, errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Sol2Evm oracle generate", "receiptHash", receiptHash)

	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.Id), uint64(m.Cfg.MapChainID))
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

	return 1, nil
}

func genReceipt(toChainId msg.ChainId, log *Log, routerOrder *RouteOrder, logData map[string]string) (*common.Hash, error) {
	orderId := common.HexToHash(logData["orderId"])
	chainAndGasLimit := common.HexToHash(logData["chainAndGasLimit"])
	gasLimit := big.NewInt(0).SetBytes(chainAndGasLimit.Bytes()[24:])
	token := make([]byte, 0)
	for _, ele := range routerOrder.FromToken {
		token = append(token, ele)
	}

	form := make([]byte, 0)
	for _, ele := range routerOrder.Payer {
		form = append(form, ele)
	}

	swapStruct, err := parseSwapData(routerOrder.SwapData)
	if err != nil {
		return nil, err
	}

	eo := MessageOutEvent{
		FromChain:   big.NewInt(int64(routerOrder.FromChainID)),
		ToChain:     big.NewInt(int64(routerOrder.ToChainID)),
		OrderId:     orderId,
		Amount:      big.NewInt(int64(routerOrder.AmountOut)),
		Token:       token,
		From:        form,
		SwapData:    routerOrder.SwapData,
		GasLimit:    gasLimit,
		Mos:         []byte(mapprotocol.MosMapping[toChainId]),
		Initiator:   swapStruct.Initiator,
		Relay:       swapStruct.Relay,
		MessageType: uint8(swapStruct.MessageType.Int64()),
		To:          swapStruct.Receiver,
	}
	data, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Pack(&eo)
	if err != nil {
		return nil, errors.Wrap(err, "marshal event code failed")
	}
	fmt.Println("log.Addr ", log.Addr)
	addr := make([]byte, 0, 64)
	for _, ele := range solana.MustPublicKeyFromBase58(log.Addr) {
		addr = append(addr, ele)
	}
	// abi
	recePack, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolPackReceipt].Inputs.Pack(addr, []byte(log.Topic), data)
	if err != nil {
		return nil, errors.Wrap(err, "marshal sol pack failed")
	}
	receipt := common.BytesToHash(crypto.Keccak256(recePack))
	return &receipt, nil
}

type MessageOutEvent struct {
	Relay       bool
	MessageType uint8
	FromChain   *big.Int
	ToChain     *big.Int
	OrderId     [32]byte
	Mos         []byte
	Token       []byte
	Initiator   []byte
	From        []byte
	To          []byte
	Amount      *big.Int
	GasLimit    *big.Int
	SwapData    []byte
}

func getSigner(log *types.Log, receiptHash common.Hash, selfId, toChainID uint64) (*chain.ProposalInfoResp, error) {
	bn := big.NewInt(int64(log.BlockNumber))
	ret, err := chain.MulSignInfo(0, selfId, toChainID)
	if err != nil {
		return nil, err
	}
	fmt.Println("Get Version ret", ret)

	piRet, err := chain.ProposalInfo(0, selfId, toChainID, bn, receiptHash, ret.Version)
	if err != nil {
		return nil, err
	}
	if !piRet.CanVerify {
		return nil, chain.NotVerifyAble
	}
	fmt.Println("ProposalInfo success", "piRet", piRet)
	return piRet, nil
}
