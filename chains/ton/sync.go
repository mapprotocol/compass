package ton

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/internal/stream"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
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
		// todo remove
		fmt.Println("sync to mpa sleeping......")
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

func filter(m *sync) (*Log, error) {
	topic := ""
	for idx, ele := range m.cfg.Event {
		topic += ele
		if idx != len(m.cfg.Event)-1 {
			topic += ","
		}
	}
	data, err := chain.Request(fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
		fmt.Sprintf("id=%d&project_id=%d&chain_id=%d&topic=%s&limit=1",
			m.Cfg.StartBlock.Int64(), 7, m.Cfg.Id, topic))) // todo project_id
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
	for _, ele := range back.List { //
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
		m.Log.Info("filter find log", "id", ele.Id, "topic", topic, "txHash", ele.TxHash)
	}

	return &ret, nil
}

func messengerHandler(m *sync) (int64, error) {
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "filter failed")
	}
	if log == nil || log.Id == 0 {
		return 0, nil
	}

	hexData, err := hex.DecodeString(log.Data)
	if err != nil {
		return 0, errors.Wrap(err, "failed to decode log data to hex")
	}

	body := &cell.Cell{}
	if err := json.Unmarshal(hexData, &body); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal log data to cell")
	}

	slice := body.BeginParse()
	messageOut, err := parseMessageOutEvent(slice)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse message out event")
	}

	receiptHash, receiptPack, err := genReceiptHash(log.Addr, log.Topic, messageOut)
	if err != nil {
		return 0, errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Ton2Evm messenger generate receipt hash", "receiptHash", receiptHash)
	proposalInfo, err := getSigner(log.BlockNumber, receiptHash, uint64(m.cfg.Id), uint64(m.cfg.MapChainID))
	if err != nil {
		return 0, err
	}

	pd := proof.SignLogData{
		ProofType:   1,
		BlockNum:    big.NewInt(log.BlockNumber),
		ReceiptRoot: receiptHash,
		Signatures:  proposalInfo.Signatures,
		Proof:       receiptPack,
	}

	input, err := mapprotocol.GetAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return 0, errors.Wrap(err, "pack getBytes failed")
	}

	orderId := common.BytesToHash(messageOut.OrderId[:]) // todo order id 如何处理？
	finalInput, err := mapprotocol.PackInput(mapprotocol.Mcs, mapprotocol.MethodOfMessageIn, new(big.Int).SetUint64(uint64(m.Cfg.Id)), big.NewInt(int64(0)), orderId, input)
	if err != nil {
		return 0, errors.Wrap(err, "pack mcs input failed")
	}

	txHash, err := Base64ToHex(log.TxHash)
	if err != nil {
		return 0, errors.Wrap(err, "failed to convert txHash to hex")
	}
	message := msg.NewSwapWithProof(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{finalInput, messageOut.OrderId, log.BlockNumber, "0x" + txHash}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return 0, nil
	}

	return log.Id, nil
}

func oracleHandler(m *sync) (int64, error) {
	log, err := filter(m)
	if err != nil {
		return 0, errors.Wrap(err, "failed to filter log")
	}
	if log == nil || log.Id == 0 {
		return 0, nil
	}
	m.Log.Info("Filter find Log", "id", log.Id, "txHash", log.TxHash)

	hexData, err := hex.DecodeString(log.Data)
	if err != nil {
		return 0, errors.Wrap(err, "failed to decode log data to hex")
	}

	body := &cell.Cell{}
	if err := json.Unmarshal(hexData, &body); err != nil {
		return 0, errors.Wrap(err, "failed to unmarshal log data to cell")
	}

	slice := body.BeginParse()
	messageOut, err := parseMessageOutEvent(slice)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse message out event")
	}

	receiptHash, _, err := genReceiptHash(log.Addr, log.Topic, messageOut)
	if err != nil {
		return 0, errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Ton2Evm oracle generate receipt hash", "receiptHash", receiptHash, "srcHash", log.TxHash)

	ret, err := chain.MulSignInfo(0, uint64(m.Cfg.MapChainID))
	if err != nil {
		return 0, errors.Wrap(err, "mul sign failed")
	}

	bn := big.NewInt(log.BlockNumber)
	input, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receiptHash, ret.Version, bn, big.NewInt(int64(m.Cfg.Id)))
	if err != nil {
		return 0, errors.Wrap(err, "oracle pack input failed")
	}

	message := msg.NewProposal(m.Cfg.Id, m.Cfg.MapChainID, []interface{}{input, &receiptHash, bn}, m.MsgCh)
	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("subscription error: failed to route message", "err", err)
		return 0, nil
	}

	return log.Id, nil
}

func genReceiptHash(addr, topic string, messageOut *MessageOutEvent) (common.Hash, []byte, error) {

	data, err := encodeMessageOutEvent(messageOut)
	if err != nil {
		return common.Hash{}, nil, errors.Wrap(err, "failed to encode message out event")
	}

	parseAddr, err := address.ParseAddr(addr)
	if err != nil {
		return common.Hash{}, nil, errors.Wrap(err, "failed to parse address")
	}

	packedProof, err := encodeProof(convertToBytes(parseAddr), common.Hex2Bytes(topic), data)
	if err != nil {
		return common.Hash{}, nil, errors.Wrap(err, "failed to encode proof")
	}

	receiptHash := common.BytesToHash(crypto.Keccak256(packedProof))
	return receiptHash, packedProof, nil
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

func match(target string, contracts []string) int {
	for idx, ele := range contracts {
		if strings.ToLower(ele) == strings.ToLower(target) {
			return idx
		}
	}
	return -1
}

func Base64ToHex(base64Str string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decodedBytes), nil
}
