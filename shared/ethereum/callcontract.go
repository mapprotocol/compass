// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/mapprotocol/compass/internal/tx"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
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

func ParseEthLogIntoSwapWithProofArgs(log types.Log, bridgeAddr common.Address, receipts []*types.Receipt, method string, fId, tId msg.ChainId) ([]byte, error) {
	// calc tx proof
	blockNumber := log.BlockNumber
	transactionIndex := log.TxIndex

	proof := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(transactionIndex)
	if err != nil {
		return nil, err
	}

	// assemble trie tree
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}
	for i, r := range receipts {
		key, err := rlp.EncodeToBytes(uint(i))
		if err != nil {
			return nil, err
		}
		value, err := rlp.EncodeToBytes(r)
		if err != nil {
			return nil, err
		}
		tr.Update(key, value)
	}

	tr = DeriveTire(receipts, tr)
	if err = tr.Prove(key, 0, proof); err != nil {
		return nil, err
	}

	txProve := mapprotocol.TxProve{
		Receipt:     receipts[transactionIndex],
		Prove:       proof.NodeList(),
		BlockNumber: blockNumber,
		TxIndex:     transactionIndex,
	}

	txProofBytes, err := rlp.EncodeToBytes(txProve)
	if err != nil {
		return nil, err
	}

	rp := mapprotocol.NewReceiptProof{
		Router:   bridgeAddr,
		Coin:     bridgeAddr, // common.BytesToAddress(token),
		SrcChain: big.NewInt(0).SetUint64(uint64(fId)),
		DstChain: big.NewInt(0).SetUint64(uint64(tId)),
		TxProve:  txProofBytes,
	}

	payloads, err := rlp.EncodeToBytes(rp)
	if err != nil {
		return nil, err
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), payloads)
	//pack, err := mapprotocol.PackInput(mapprotocol.Near, mapprotocol.MethodVerifyProofData, payloads)
	if err != nil {
		return nil, errors.Wrap(err, "transferIn pack failed")
	}

	return pack, nil
}

type MapTxProve struct {
	Header      *maptypes.Header
	Receipt     *types.Receipt
	Prove       light.NodeList
	BlockNumber uint64
	TxIndex     uint
}

func GetProof(client *ethclient.Client, latestBlock *big.Int, log *types.Log, method string, fId msg.ChainId) ([]byte, error) {
	header, err := client.MAPHeaderByNumber(context.Background(), latestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to query header Logs: %w", err)
	}
	txsHash, err := mapprotocol.GetMapTransactionsHashByBlockNumber(client, latestBlock)
	if err != nil {
		return nil, fmt.Errorf("idSame unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(client, txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	//
	remainder := big.NewInt(0).Mod(latestBlock, big.NewInt(mapprotocol.EpochOfMap))
	if remainder.Cmp(mapprotocol.Big0) == 0 {
		lr, err := mapprotocol.GetLastReceipt(client, latestBlock)
		if err != nil {
			return nil, fmt.Errorf("unable to get last receipts in epoch last %w", err)
		}
		receipts = append(receipts, lr)
	}
	_, data, err := AssembleMapProof(client, *log, receipts, header, fId, method, "")
	return data, err
}

func AssembleMapProof(cli *ethclient.Client, log types.Log, receipts []*types.Receipt,
	header *maptypes.Header, fId msg.ChainId, method, zkUrl string) (uint64, []byte, error) {
	toChainID := log.Topics[2]
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])
	txIndex := log.TxIndex
	aggPK, ist, aggPKBytes, err := mapprotocol.GetAggPK(cli, new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return 0, nil, err
	}

	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	proof, err := iproof.Get(receipts, txIndex)
	if err != nil {
		return 0, nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := util.Key2Hex(key, len(proof))
	if name, ok := mapprotocol.OnlineChaId[msg.ChainId(97)]; ok && strings.ToLower(name) != "near" {
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
			Proof:    proof,
			Ist:      *istanbulExtra,
			TxReceiptRlp: mapprotocol.TxReceiptRlp{
				ReceiptType: receipt.ReceiptType,
				ReceiptRlp:  nrRlp,
			},
		}

		var pack []byte
		if zkUrl == "" {
			pack, err = mapprotocol.Map2Other.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(rp)
			if err != nil {
				return 0, nil, errors.Wrap(err, "getBytes failed")
			}
		} else {
			zkProof, err := mapprotocol.GetZkProof(zkUrl, fId, header.Number.Uint64())
			if err != nil {
				return 0, nil, errors.Wrap(err, "GetZkProof failed")
			}

			pack, err = mapprotocol.Mcs.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(rp, zkProof)
			if err != nil {
				return 0, nil, errors.Wrap(err, "getBytes failed")
			}
		}

		payloads, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), pack)
		//payloads, err := mapprotocol.PackInput(mapprotocol.Other, mapprotocol.MethodVerifyProofData, pack)
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

	nProof := make([]string, 0, len(proof))
	for _, p := range proof {
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
