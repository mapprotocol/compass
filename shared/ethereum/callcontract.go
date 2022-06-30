// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var (
	ZeroAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	// SwapIn function swapIn(bytes32 hash, address token, address from, address to, uint amount, uint fromChainID,uint toChainID)
	SwapIn = "swapIn(bytes32,address,address,address,uint256,uint256,uint256,address,bytes)"

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

func ComposeMsgPayloadWithSignature(sig string, msgPayload []interface{}) []byte {
	// signature
	sigbytes := crypto.Keccak256Hash([]byte(sig))

	var data []byte
	data = append(data, sigbytes[:4]...)
	data = append(data, msgPayload[0].([]byte)...)
	return data
}

func ParseEthLogIntoSwapArgs(log types.Log, bridgeAddr common.Address) (uint64, uint64, []byte, error) {
	token := log.Topics[1].Bytes()
	from := log.Topics[2].Bytes()
	to := log.Topics[3].Bytes()
	// every 32 bytes forms a value
	var orderHash [32]byte
	copy(orderHash[:], log.Data[:32])
	amount := log.Data[32:64]

	fromChainID := log.Data[64:96]
	toChainID := log.Data[96:128]
	uFromChainID := binary.BigEndian.Uint64(fromChainID[len(fromChainID)-8:])
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])

	payloads, err := SwapInArgs.Pack(
		orderHash,
		common.BytesToAddress(token),
		common.BytesToAddress(from),
		common.BytesToAddress(to),
		big.NewInt(0).SetBytes(amount),
		big.NewInt(0).SetBytes(fromChainID),
		big.NewInt(0).SetBytes(toChainID),
		bridgeAddr,
		[]byte{},
	)
	if err != nil {
		return 0, 0, nil, err
	}
	return uFromChainID, uToChainID, payloads, nil
}

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

func ParseEthLogIntoSwapWithProofArgs(log types.Log, bridgeAddr common.Address, receipts []*types.Receipt) (uint64, uint64, []byte, error) {
	token := log.Topics[1].Bytes()
	from := log.Topics[2].Bytes()
	to := log.Topics[3].Bytes()
	// every 32 bytes forms a value
	var orderHash [32]byte
	copy(orderHash[:], log.Data[:32])
	amount := log.Data[32:64]

	fromChainID := log.Data[64:96]
	toChainID := log.Data[96:128]
	uFromChainID := binary.BigEndian.Uint64(fromChainID[len(fromChainID)-8:])
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])

	// calc tx proof
	blockNumber := log.BlockNumber
	transactionIndex := log.TxIndex

	proof := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(transactionIndex)
	if err != nil {
		return 0, 0, nil, err
	}

	// assemble trie tree
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return 0, 0, nil, err
	}
	for i, r := range receipts {
		key, err := rlp.EncodeToBytes(uint(i))
		if err != nil {
			return 0, 0, nil, err
		}
		value, err := rlp.EncodeToBytes(r)
		if err != nil {
			return 0, 0, nil, err
		}
		tr.Update(key, value)
	}

	tr = DeriveTire(receipts, tr)
	if err = tr.Prove(key, 0, proof); err != nil {
		return 0, 0, nil, err
	}

	txProve := TxProve{
		Tx: &TxParams{
			From:  from,
			To:    to,
			Value: big.NewInt(0).SetBytes(amount),
		},
		Receipt:     receipts[transactionIndex],
		Prove:       proof.NodeList(),
		BlockNumber: blockNumber,
		TxIndex:     transactionIndex,
	}

	txProofBytes, err := rlp.EncodeToBytes(txProve)
	if err != nil {
		return 0, 0, nil, err
	}

	payloads, err := SwapInArgs.Pack(
		orderHash,
		common.BytesToAddress(token),
		common.BytesToAddress(from),
		common.BytesToAddress(to),
		big.NewInt(0).SetBytes(amount),
		big.NewInt(0).SetBytes(fromChainID),
		big.NewInt(0).SetBytes(toChainID),
		bridgeAddr,
		txProofBytes,
	)
	if err != nil {
		return 0, 0, nil, err
	}
	return uFromChainID, uToChainID, payloads, nil
}

type MapTxProve struct {
	Header      *maptypes.Header
	Receipt     *types.Receipt
	Prove       light.NodeList
	BlockNumber uint64
	TxIndex     uint
}

func ParseMapLogIntoSwapWithMapProofArgs(cli *ethclient.Client, log types.Log, bridgeAddr common.Address, receipts []*types.Receipt, header *maptypes.Header) (uint64, uint64, []byte, error) {
	fromChainID := log.Data[64:96]
	toChainID := log.Data[96:128]
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
	rp := mapprotocol.ReceiptProof{
		Header:   mapprotocol.ConvertHeader(header),
		AggPk:    aggPK,
		Receipt:  receipt,
		KeyIndex: rlp.AppendUint64(key[:0], uint64(txIndex)),
		Proof:    proof,
	}
	payloads, err := mapprotocol.PackLightNodeInput(mapprotocol.MethodVerifyProofData, rp)
	if err != nil {
		return 0, 0, nil, err
	}
	return uFromChainID, uToChainID, payloads, nil
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
