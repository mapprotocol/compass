package proof

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/golang/groupcache/lru"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/pkg/errors"
)

var (
	CacheReceipt = lru.New(30)
)

type ReceiptRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             types.Bloom
	Logs              []*types.Log
}

type Data struct {
	BlockNum     *big.Int
	ReceiptProof ReceiptProof
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

type NewData struct {
	BlockNum     *big.Int
	ReceiptProof NewReceiptProof
}

type SignData struct {
	BlockNum     *big.Int
	ReceiptRoot  [32]byte
	Signatures   [][]byte
	ReceiptProof NewReceiptProof
}

type SignLogData struct {
	ProofType   uint8
	BlockNum    *big.Int
	ReceiptRoot [32]byte
	Signatures  [][]byte
	Proof       []byte
}

type NewReceiptProof struct {
	TxReceipt   []byte
	ReceiptType *big.Int
	KeyIndex    []byte
	Proof       [][]byte
}

var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func Get(receipts DerivableList, txIndex uint) ([][]byte, error) {
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

func DeriveTire(rs DerivableList, tr *trie.Trie) *trie.Trie {
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

type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	return common.CopyBytes(buf.Bytes())
}

func Pack(fId msg.ChainId, method string, abi abi.ABI, params ...interface{}) ([]byte, error) {
	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(params...)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func Oracle(blockNumber uint64, receipt *mapprotocol.TxReceipt, key []byte, prf [][]byte, fId msg.ChainId, method string, idx int,
	abi abi.ABI, orderId [32]byte, map2other bool) ([]byte, error) {
	nr := mapprotocol.MapTxReceipt{
		PostStateOrStatus: receipt.PostStateOrStatus,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             receipt.Bloom,
		Logs:              receipt.Logs,
	}
	nrRlp, err := rlp.EncodeToBytes(nr)
	if err != nil {
		return nil, err
	}

	pd := NewData{
		BlockNum: big.NewInt(int64(blockNumber)),
		ReceiptProof: NewReceiptProof{
			TxReceipt:   nrRlp,
			ReceiptType: receipt.ReceiptType,
			KeyIndex:    util.Key2Hex(key, len(prf)),
			Proof:       prf,
		},
	}

	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	var ret []byte
	ret, err = mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)), big.NewInt(int64(idx)),
		orderId, input)

	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func Completion(bytes []byte, number int) []byte {
	ret := make([]byte, 0, number)
	for i := 0; i < number-len(bytes); i++ {
		ret = append(ret, byte(0))
	}
	ret = append(ret, bytes...)
	return ret
}

func log2Proof(log *types.Log) []byte {
	ret := make([]byte, 0)
	ret = append(ret, log.Address.Bytes()...)
	ret = append(ret, []byte{0, 0, 0, 0}...)
	ret = append(ret, Completion(big.NewInt(int64(len(log.Topics))).Bytes(), 4)...)
	ret = append(ret, Completion(big.NewInt(int64(len(log.Data))).Bytes(), 4)...)
	for _, tp := range log.Topics {
		ret = append(ret, tp.Bytes()...)
	}
	ret = append(ret, log.Data...)
	return ret
}

func SignOracle(header *maptypes.Header, receipt *mapprotocol.TxReceipt, key []byte, prf [][]byte, fId msg.ChainId,
	idx int, method string, sign [][]byte, log *types.Log, proofType int64) ([]byte, error) {
	var (
		fixedHash   [32]byte
		pt          = uint8(0)
		newPrf      = make([]byte, 0)
		blockNumber = big.NewInt(0).SetUint64(log.BlockNumber)
	)
	switch proofType {
	case constant.ProofTypeOfNewOracle:
		nr := mapprotocol.MapTxReceipt{
			PostStateOrStatus: receipt.PostStateOrStatus,
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom,
			Logs:              receipt.Logs,
		}
		nrRlp, err := rlp.EncodeToBytes(nr)
		if err != nil {
			return nil, err
		}

		for i, v := range header.ReceiptHash {
			fixedHash[i] = v
		}
		if receipt.ReceiptType.Int64() != 0 {
			n := make([]byte, 0)
			n = append(n, receipt.ReceiptType.Bytes()...)
			n = append(n, nrRlp...)
			nrRlp = n
		}

		rpf := NewReceiptProof{
			TxReceipt:   nrRlp,
			ReceiptType: receipt.ReceiptType,
			KeyIndex:    util.Key2Hex(key, len(prf)),
			Proof:       prf,
		}

		newPrf, err = mapprotocol.PackAbi.Methods[mapprotocol.MethodOfMptPack].Inputs.Pack(rpf)
		if err != nil {
			return nil, err
		}
	case constant.ProofTypeOfLogOracle:
		pt = 1
		newPrf = log2Proof(log)
		logIdx := log.Index
		blockNumber = GenLogBlockNumber(blockNumber, logIdx)
		fixedHash = common.BytesToHash(crypto.Keccak256(newPrf))
	default:
		return nil, errors.New("invalid proof type")
	}

	pd := SignLogData{
		ProofType:   pt,
		BlockNum:    blockNumber,
		ReceiptRoot: fixedHash,
		Signatures:  sign,
		Proof:       newPrf,
	}

	input, err := mapprotocol.GetAbi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	ret, err := mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)),
		big.NewInt(int64(idx)), log.Topics[1], input)
	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func V3Pack(fId msg.ChainId, method string, abi abi.ABI, idx int, orderId [32]byte, map2other bool, params ...interface{}) ([]byte, error) {
	input, err := abi.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(params...)
	if err != nil {
		return nil, errors.Wrap(err, "pack getBytes failed")
	}

	var ret []byte
	ret, err = mapprotocol.PackInput(mapprotocol.Mcs, method, big.NewInt(0).SetUint64(uint64(fId)),
		big.NewInt(int64(idx)), orderId, input)

	if err != nil {
		return nil, errors.Wrap(err, "pack mcs input failed")
	}

	return ret, nil
}

func GenLogBlockNumber(bn *big.Int, idx uint) *big.Int {
	ret := make([]byte, 0, 28)
	ret = append(ret, Completion(big.NewInt(int64(idx)).Bytes(), 4)...)
	ret = append(ret, Completion(bn.Bytes(), 8)...)
	return big.NewInt(0).SetBytes(ret)
}
