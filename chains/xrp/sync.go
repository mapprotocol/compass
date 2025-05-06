package xrp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/internal/abi"
	"github.com/mapprotocol/compass/internal/contract"
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
	LogHandler func(*sync, *stream.GetMosResp) error
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

			m.Cfg.StartBlock = big.NewInt(id)
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

		uri := fmt.Sprintf("%s/%s?%s", m.Cfg.FilterHost, constant.FilterUrl,
			fmt.Sprintf("id=%d&chain_id=%d&topic=%s&limit=1",
				m.Cfg.StartBlock.Int64(), m.Cfg.Id, topic))
		data, err := chain.Request(uri)
		if err != nil {
			return 0, err
		}
		listData, err := json.Marshal(data)
		if err != nil {
			return 0, errors.Wrap(err, "marshal resp.Data failed")
		}
		back := stream.MosListResp{}
		err = json.Unmarshal(listData, &back)
		if err != nil {
			return 0, err
		}
		if len(back.List) == 0 {
			return 0, nil
		}

		retId := int64(0)
		for _, ele := range back.List {
			m.Log.Info("Xrp find a log", "id", ele.Id, "block", ele.BlockNumber)
			log := ele
			err = lh(m, log)
			if err != nil {
				return 0, err
			}
			retId = ele.Id
		}

		return retId, nil
	}
}

func mos(m *sync, log *stream.GetMosResp) error {
	receiptHash, receiptPack, err := genReceipt(log)
	if err != nil {
		return errors.Wrap(err, "gen receipt failed")
	}
	m.Log.Info("Xrp2Evm mos generate", "receiptHash", receiptHash)
	bn := proof.GenLogBlockNumber(big.NewInt(int64(log.BlockNumber)), log.LogIndex)
	proposalInfo, err := chain.GetSigner(bn, *receiptHash, uint64(m.cfg.Id), uint64(m.cfg.MapChainID))
	if err != nil {
		return err
	}
	var fixedHash [32]byte
	for i, v := range receiptHash {
		fixedHash[i] = v
	}
	pd := proof.SignLogData{
		ProofType:   constant.ProofTypeOfContract,
		BlockNum:    bn,
		ReceiptRoot: fixedHash,
		Signatures:  proposalInfo.Signatures,
		Proof:       receiptPack,
	}

	input, err := mapprotocol.GetAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return errors.Wrap(err, "pack getBytes failed")
	}

	orderIdStr := strings.Split(log.Topic, ",")[1]
	orderId := common.HexToHash(orderIdStr)
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

func oracle(m *sync, log *stream.GetMosResp) error {
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
		version = append(version, v)
	}

	bn := proof.GenLogBlockNumber(big.NewInt(int64(log.BlockNumber)), log.LogIndex)
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

func genReceipt(log *stream.GetMosResp) (*common.Hash, []byte, error) {
	orderIdStr := strings.Split(log.Topic, ",")[1]
	// decode log data
	hex2Bytes := common.Hex2Bytes(strings.TrimPrefix(log.LogData, "0x"))
	if len(hex2Bytes) <= 0 {
		return nil, nil, errors.New("invalid log data")
	}
	values, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Unpack(hex2Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal event code failed")
	}
	event := mapprotocol.MessageOutEvent{}
	err = mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Copy(&event, values)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unmarshal event code failed")
	}

	orderId := common.HexToHash(orderIdStr)
	bridgeParam, err := abi.DecodeBridgeParam(event.SwapData)
	if err != nil {
		return nil, nil, errors.Wrap(err, "decode bridge param failed")
	}

	if len(bridgeParam.SwapData) > 0 {
		// check swapData
		pass, err := contract.Validate(event.Relay, event.ToChain, event.Amount, event.Token, event.To, bridgeParam.SwapData)
		if err != nil {
			return nil, nil, err
		}
		if !pass {
			return nil, nil, fmt.Errorf("invalid swapData (%s)", bridgeParam.SwapData)
		}
	}

	eo := mapprotocol.MessageOutEvent{
		FromChain:   event.FromChain,
		ToChain:     event.ToChain,
		OrderId:     orderId,
		Amount:      event.Amount,
		Token:       event.Token,
		From:        event.From, // Xrp address
		SwapData:    bridgeParam.SwapData,
		GasLimit:    event.GasLimit,
		Mos:         event.Mos,
		Initiator:   event.Initiator, // Xrp address
		Relay:       event.Relay,
		MessageType: event.MessageType,
		To:          event.To,
	}
	data, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolEventEncode].Inputs.Pack(&eo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal event code failed")
	}
	// abi
	receiptPack, err := mapprotocol.SolAbi.Methods[mapprotocol.MethodOfSolPackReceipt].Inputs.Pack(
		common.Hex2Bytes(strings.TrimPrefix(log.ContractAddress, "0x")),
		common.Hex2Bytes(strings.TrimPrefix(log.Topic, "0x")), data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal pack failed")
	}
	receipt := common.BytesToHash(crypto.Keccak256(receiptPack))
	return &receipt, receiptPack, nil
}
