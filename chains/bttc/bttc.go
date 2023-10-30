package bttc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/klaytn/klaytn/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

type BlockHeader struct {
	ParentHash       []byte         `json:"parentHash"`
	Sha3Uncles       []byte         `json:"sha3Uncles"`
	Miner            common.Address `json:"miner"`
	StateRoot        []byte         `json:"stateRoot"`
	TransactionsRoot []byte         `json:"transactionsRoot"`
	ReceiptsRoot     []byte         `json:"receiptsRoot"`
	LogsBloom        []byte         `json:"logsBloom"`
	Difficulty       *big.Int       `json:"difficulty"`
	Number           *big.Int       `json:"number"`
	GasLimit         *big.Int       `json:"gasLimit"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        *big.Int       `json:"timestamp"`
	ExtraData        []byte         `json:"extraData"`
	MixHash          []byte         `json:"mixHash"`
	Nonce            []byte         `json:"nonce"`
}

func convertHeader(header *types.Header) BlockHeader {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}
	return BlockHeader{
		ParentHash:       util.HashToByte(header.ParentHash),
		Sha3Uncles:       util.HashToByte(header.UncleHash),
		Miner:            utils.ZeroAddress,
		StateRoot:        util.HashToByte(header.Root),
		TransactionsRoot: util.HashToByte(header.TxHash),
		ReceiptsRoot:     util.HashToByte(header.ReceiptHash),
		LogsBloom:        bloom,
		Difficulty:       header.Difficulty,
		Number:           header.Number,
		GasLimit:         new(big.Int).SetUint64(header.GasLimit),
		GasUsed:          new(big.Int).SetUint64(header.GasUsed),
		Timestamp:        new(big.Int).SetUint64(header.Time),
		ExtraData:        header.Extra,
		MixHash:          util.HashToByte(header.MixDigest),
		Nonce:            nonce,
	}
}

type ProofData struct {
	Headers      []BlockHeader
	ReceiptProof ReceiptProof
}

type ReceiptProof struct {
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(headers []BlockHeader, log types.Log, fId msg.ChainId, allR []*types.Receipt, cullSys []*types.Receipt, method string) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(allR[txIndex])
	if err != nil {
		return nil, err
	}

	proof, err := getProof(cullSys, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := util.OtherKey2Hex(key)

	pd := ProofData{
		Headers: headers,
		ReceiptProof: ReceiptProof{
			TxReceipt: *receipt,
			KeyIndex:  ek,
			Proof:     proof,
		},
	}

	//fmt.Println("keyIndex ----------- ", "0x"+common.Bytes2Hex(pd.ReceiptProof.KeyIndex))
	//printProof(pd.ReceiptProof.Proof)
	//printReceipt(&pd.ReceiptProof.TxReceipt)
	//for _, bk := range pd.Headers {
	//	printHeader(bk)
	//}

	input, err := mapprotocol.Bttc.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}
	// fmt.Println("bttc -------- input", "0x"+common.Bytes2Hex(input))
	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	//pack, err := mapprotocol.Bttc.Pack(mapprotocol.MethodVerifyProofData, input)
	if err != nil {
		return nil, err
	}

	return pack, nil
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = utils.DeriveTire(receipts, tr)
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
