// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapo

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/mapprotocol/compass/internal/arb"
	"github.com/mapprotocol/compass/internal/constant"
	"math/big"
	"strings"
	"sync"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

type TxParams struct {
	From  []byte
	To    []byte
	Value *big.Int
}

type TxProve struct {
	Tx          *TxParams
	Receipt     *types.Receipt
	Prove       light.NodeList
	BlockNumber uint64
	TxIndex     uint
}

func AssembleEthProof(log *types.Log, receipts []*types.Receipt, method string, fId msg.ChainId, proofType int64) ([]byte, error) {
	var payloads []byte
	switch proofType {
	case constant.ProofTypeOfOrigin:
	case constant.ProofTypeOfZk:
	case constant.ProofTypeOfOracle:
		receipt, err := mapprotocol.GetTxReceipt(receipts[log.TxIndex])
		if err != nil {
			return nil, err
		}

		pr := arb.Receipts{}
		for _, r := range receipts {
			pr = append(pr, &arb.Receipt{Receipt: r})
		}

		prf, err := proof.Get(pr, log.TxIndex)
		if err != nil {
			return nil, err
		}

		var key []byte
		key = rlp.AppendUint64(key[:0], uint64(log.TxIndex))

		pd := struct {
			BlockNum     *big.Int
			ReceiptProof ReceiptProof
		}{
			BlockNum: big.NewInt(int64(log.BlockNumber)),
			ReceiptProof: ReceiptProof{
				TxReceipt: *receipt,
				KeyIndex:  util.Key2Hex(key, len(prf)),
				Proof:     prf,
			},
		}

		printProof(pd.ReceiptProof.Proof)

		payloads, err = mapprotocol.OracleAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
		if err != nil {
			return nil, err
		}
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), payloads)
	if err != nil {
		return nil, errors.Wrap(err, "mcs input pack failed")
	}

	//fmt.Println("-------- ", "0x"+common.Bytes2Hex(pack))
	return pack, nil
}

func printProof(proof [][]byte) {
	p := make([]string, 0, len(proof))
	for _, v := range proof {
		p = append(p, "0x"+common.Bytes2Hex(v))
		//fmt.Println("============================== proof: ", "0x"+common.Bytes2Hex(v))
	}
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleMapProof(cli *ethclient.Client, log *types.Log, receipts []*types.Receipt,
	header *maptypes.Header, fId msg.ChainId, method, zkUrl string, proofType int64) (uint64, []byte, error) {
	toChainID := log.Topics[2]
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])
	txIndex := log.TxIndex
	aggPK, ist, aggPKBytes, err := mapprotocol.GetAggPK(cli, new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return 0, nil, err
	}

	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	prf, err := proof.Get(types.Receipts(receipts), txIndex)
	if err != nil {
		return 0, nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := Key2Hex(key, len(prf))
	if uToChainID == 8217 {
		ek = util.Key2Hex(key, len(prf))
	}
	if zkUrl != "" {
		ek = util.Key2Hex(key, len(prf))
	}
	if name, ok := mapprotocol.OnlineChaId[msg.ChainId(uToChainID)]; ok && strings.ToLower(name) != "near" {
		istanbulExtra := mapprotocol.ConvertIstanbulExtra(ist)
		nr := mapprotocol.MapTxReceipt{
			PostStateOrStatus: receipt.PostStateOrStatus,
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom,
			Logs:              receipt.Logs,
		}

		nrRlp, err := rlp.EncodeToBytes(nr)
		if err != nil {
			return 0, nil, err
		}
		rp := mapprotocol.NewMapReceiptProof{
			Header:   mapprotocol.ConvertHeader(header),
			AggPk:    aggPK,
			KeyIndex: ek,
			Proof:    prf,
			Ist:      *istanbulExtra,
			TxReceiptRlp: mapprotocol.TxReceiptRlp{
				ReceiptType: receipt.ReceiptType,
				ReceiptRlp:  nrRlp,
			},
		}

		var pack []byte
		switch proofType {
		case constant.ProofTypeOfOrigin:
			pack, err = mapprotocol.Map2Other.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(rp)
			if err != nil {
				return 0, nil, errors.Wrap(err, "getBytes failed")
			}
		case constant.ProofTypeOfZk:
			zkProof, err := mapprotocol.GetZkProof(zkUrl, fId, header.Number.Uint64())
			if err != nil {
				return 0, nil, errors.Wrap(err, "GetZkProof failed")
			}

			pack, err = mapprotocol.Mcs.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(rp, zkProof)
			if err != nil {
				return 0, nil, errors.Wrap(err, "getBytes failed")
			}
		case constant.ProofTypeOfOracle:
			pd := struct {
				BlockNum     *big.Int
				ReceiptProof ReceiptProof
			}{
				BlockNum: header.Number,
				ReceiptProof: ReceiptProof{
					TxReceipt: *receipt,
					KeyIndex:  util.Key2Hex(key, len(prf)),
					Proof:     prf,
				},
			}

			pack, err = mapprotocol.OracleAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
			if err != nil {
				return 0, nil, errors.Wrap(err, "getBytes failed")
			}
		}

		payloads, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), pack)
		if err != nil {
			return 0, nil, errors.Wrap(err, "eth pack failed")
		}

		return uToChainID, payloads, nil
	}

	bytesBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(bytesBuffer, binary.LittleEndian, uint64(txIndex))
	if err != nil {
		return 0, nil, err
	}

	nProof := make([]string, 0, len(prf))
	for _, p := range prf {
		nProof = append(nProof, "0x"+common.Bytes2Hex(p))
	}
	m := map[string]interface{}{
		"header": mapprotocol.ConvertNearNeedHeader(header),
		"agg_pk": map[string]interface{}{
			"xr": "0x" + common.Bytes2Hex(aggPKBytes[32:64]),
			"xi": "0x" + common.Bytes2Hex(aggPKBytes[:32]),
			"yi": "0x" + common.Bytes2Hex(aggPKBytes[64:96]),
			"yr": "0x" + common.Bytes2Hex(aggPKBytes[96:128]),
		},
		"key_index": "0x" + common.Bytes2Hex(key),
		"receipt":   ConvertNearReceipt(receipt),
		"proof":     nProof,
	}

	idx := 0
	match := false
	for lIdx, l := range receipt.Logs {
		for _, topic := range l.Topics {
			if common.BytesToHash(topic) == log.Topics[0] {
				idx = lIdx
				match = true
				break
			}
		}
		if match {
			break
		}
	}
	data, _ := json.Marshal(map[string]interface{}{
		"receipt_proof": m,
		"index":         idx,
	})
	return uToChainID, data, nil
}

func Key2Hex(str []byte, proofLength int) []byte {
	ret := make([]byte, 0)
	if len(ret)+1 == proofLength {
		ret = append(ret, str...)
	} else {
		for _, b := range str {
			ret = append(ret, b/16)
			ret = append(ret, b%16)
		}
	}
	return ret
}

type TxReceipt struct {
	ReceiptType       string  `json:"receipt_type"`
	PostStateOrStatus string  `json:"post_state_or_status"`
	CumulativeGasUsed string  `json:"cumulative_gas_used"`
	Bloom             string  `json:"bloom"`
	Logs              []TxLog `json:"logs"`
}

type TxLog struct {
	Address common.Address `json:"address"`
	Topics  []string       `json:"topics"`
	Data    string         `json:"data"`
}

func ConvertNearReceipt(h *mapprotocol.TxReceipt) *TxReceipt {
	logs := make([]TxLog, 0, len(h.Logs))
	for _, log := range h.Logs {
		topics := make([]string, 0, len(log.Topics))
		for _, t := range log.Topics {
			topics = append(topics, "0x"+common.Bytes2Hex(t))
		}
		logs = append(logs, TxLog{
			Address: log.Addr,
			Topics:  topics,
			Data:    "0x" + common.Bytes2Hex(log.Data),
		})
	}
	return &TxReceipt{
		ReceiptType:       h.ReceiptType.String(),
		PostStateOrStatus: "0x" + common.Bytes2Hex(h.PostStateOrStatus),
		CumulativeGasUsed: h.CumulativeGasUsed.String(),
		Bloom:             "0x" + common.Bytes2Hex(h.Bloom),
		Logs:              logs,
	}
}

// deriveBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// DerivableList is the input to DeriveSha.
// It is implemented by the 'Transactions' and 'Receipts' types.
// This is internal, do not use these methods.
type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	// It's really unfortunate that we need to do perform this copy.
	// StackTrie holds onto the values until Hash is called, so the values
	// written to it must not alias.
	return common.CopyBytes(buf.Bytes())
}

func DeriveTire(rs types.Receipts, tr *trie.Trie) *trie.Trie {
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	var indexBuf []byte
	for i := 1; i < rs.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	if rs.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(rs, 0, valueBuf)
		tr.Update(indexBuf, value)
	}
	for i := 0x80; i < rs.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	return tr
}
