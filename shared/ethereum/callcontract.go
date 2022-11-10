// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")

	bytesTy, _   = abi.NewType("bytes", "", nil)
	bytes32Ty, _ = abi.NewType("bytes32", "", nil)
	uint256Ty, _ = abi.NewType("uint256", "", nil)
	addressTy, _ = abi.NewType("address", "", nil)

	SwapInArgs = abi.Arguments{
		{
			Name: "_hash",
			Type: bytes32Ty,
		},
		{
			Name: "_token",
			Type: addressTy,
		},
		{
			Name: "_from",
			Type: addressTy,
		},
		{
			Name: "_to",
			Type: addressTy,
		},
		{
			Name: "_amount",
			Type: uint256Ty,
		},
		{
			Name: "_fromChainID",
			Type: uint256Ty,
		},
		{
			Name: "_toChainID",
			Type: uint256Ty,
		},
		{
			Name: "_router",
			Type: addressTy,
		},
		{
			Name: "_txProof",
			Type: bytesTy,
		},
	}
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
	//pack, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodVerifyProofData,
	//	big.NewInt(0).SetUint64(uint64(fId)), payloads)
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

func ParseMapLogIntoSwapWithProofArgsV2(cli *ethclient.Client, log types.Log, receipts []*types.Receipt,
	header *maptypes.Header) (uint64, uint64, []byte, error) {
	fromChainID := log.Data[96:128]
	toChainID := log.Data[128:160]
	uFromChainID := binary.BigEndian.Uint64(fromChainID[len(fromChainID)-8:])
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])
	txIndex := log.TxIndex
	aggPK, err := mapprotocol.GetAggPK(cli, new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return 0, 0, nil, err
	}

	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	proof, err := getProof(receipts, txIndex)
	if err != nil {
		return 0, 0, nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := Key2Hex(key, len(proof))
	if _, ok := chains.NearChainId[msg.ChainId(uToChainID)]; !ok {
		rp := mapprotocol.ReceiptProof{
			Header:   mapprotocol.ConvertHeader(header),
			AggPk:    aggPK,
			Receipt:  receipt,
			KeyIndex: ek,
			Proof:    proof,
		}

		pack, err := mapprotocol.Mcs.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(rp)
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "getBytes failed")
		}
		payloads, err := mapprotocol.PackInput(mapprotocol.Mcs, mapprotocol.MethodOfTransferIn, big.NewInt(0).SetUint64(uFromChainID), pack)
		if err != nil {
			return 0, 0, nil, errors.Wrap(err, "eth pack failed")
		}

		return uFromChainID, uToChainID, payloads, nil
	}

	bytesBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(bytesBuffer, binary.LittleEndian, uint64(txIndex))
	if err != nil {
		return 0, 0, nil, err
	}

	nProof := make([]string, 0, len(proof))
	for _, p := range proof {
		nProof = append(nProof, "0x"+common.Bytes2Hex(p))
	}
	m := map[string]interface{}{
		"header": mapprotocol.ConvertNearNeedHeader(header),
		"agg_pk": map[string]interface{}{
			"xr": "0x" + common.Bytes2Hex(aggPK.Xr.Bytes()),
			"xi": "0x" + common.Bytes2Hex(aggPK.Xi.Bytes()),
			"yi": "0x" + common.Bytes2Hex(aggPK.Yi.Bytes()),
			"yr": "0x" + common.Bytes2Hex(aggPK.Yr.Bytes()),
		},
		"key_index": "0x" + common.Bytes2Hex(key),
		"receipt":   ConvertNearReceipt(receipt),
		"proof":     nProof,
	}

	data, _ := json.Marshal(map[string]interface{}{
		"receipt_proof": m,
		"index":         log.Index,
	})
	//fmt.Printf("map2near message %v \n", string(data))
	return uFromChainID, uToChainID, data, nil
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
	ReceiptType       *big.Int `json:"receipt_type"`
	PostStateOrStatus string   `json:"post_state_or_status"`
	CumulativeGasUsed *big.Int `json:"cumulative_gas_used"`
	Bloom             string   `json:"bloom"`
	Logs              []TxLog  `json:"logs"`
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
		ReceiptType:       h.ReceiptType,
		PostStateOrStatus: "0x" + common.Bytes2Hex(h.PostStateOrStatus),
		CumulativeGasUsed: h.CumulativeGasUsed,
		Bloom:             "0x" + common.Bytes2Hex(h.Bloom),
		Logs:              logs,
	}
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}

	return proof, nil
}

/****** below is some code from atlas/core/types/hashing.go ******/

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
