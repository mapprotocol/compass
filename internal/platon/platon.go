package platon

import (
	"math/big"

	"github.com/mapprotocol/compass/pkg/ethclient"

	"github.com/mapprotocol/compass/pkg/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

type BlockHeader struct {
	ParentHash       []byte
	Miner            common.Address
	StateRoot        []byte
	TransactionsRoot []byte
	ReceiptsRoot     []byte
	LogsBloom        []byte
	Number           *big.Int
	GasLimit         *big.Int
	GasUsed          *big.Int
	Timestamp        *big.Int
	ExtraData        []byte
	Nonce            []byte
}

type UpdateBlock struct {
	Header     *BlockHeader
	Validators []ethclient.Validator
	Certs      []ethclient.QuorumCert
}

func ConvertHeader(header *types.Header) *BlockHeader {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}

	return &BlockHeader{
		ParentHash:       util.HashToByte(header.ParentHash),
		Miner:            header.Coinbase,
		StateRoot:        util.HashToByte(header.Root),
		TransactionsRoot: util.HashToByte(header.TxHash),
		ReceiptsRoot:     util.HashToByte(header.ReceiptHash),
		LogsBloom:        bloom,
		Number:           header.Number,
		GasLimit:         new(big.Int).SetUint64(header.GasLimit),
		GasUsed:          new(big.Int).SetUint64(header.GasUsed),
		Timestamp:        new(big.Int).SetUint64(header.Time),
		ExtraData:        header.Extra,
		Nonce:            nonce,
	}
}

type ProofData struct {
	ReceiptProof *ReceiptProof
	Header       *BlockHeader
	QuorumCert   *QuorumCert
}

type QuorumCert struct {
	BlockHash           [32]byte `json:"blockHash"`
	BlockIndex          *big.Int `json:"blockIndex"`
	BlockNumber         *big.Int `json:"blockNumber"`
	Epoch               *big.Int `json:"epoch"`
	ViewNumber          *big.Int `json:"viewNumber"`
	Signature           []byte   `json:"signature"`
	ValidatorSignBitMap *big.Int
	SignedCount         *big.Int
	// ValidatorSet        []byte   `json:"validatorSet"`
}

type ReceiptProof struct {
	TxReceipt *mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(block *UpdateBlock, log types.Log, receipts []*types.Receipt, method string, fId msg.ChainId) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	proof, err := iproof.Get(receipts, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := utils.Key2Hex(key, len(proof))

	rp := &ReceiptProof{
		TxReceipt: receipt,
		KeyIndex:  ek,
		Proof:     proof,
	}

	cert := &QuorumCert{
		BlockHash:           [32]byte{},
		BlockIndex:          big.NewInt(0),
		BlockNumber:         big.NewInt(0),
		Epoch:               big.NewInt(0),
		ViewNumber:          big.NewInt(0),
		Signature:           make([]byte, 0),
		ValidatorSignBitMap: big.NewInt(0),
		SignedCount:         big.NewInt(0),
	}
	if len(block.Certs) > 0 {
		cert = &QuorumCert{
			BlockHash:           common.HexToHash(block.Certs[0].BlockHash),
			BlockIndex:          big.NewInt(block.Certs[0].BlockIndex),
			BlockNumber:         big.NewInt(block.Certs[0].BlockNumber),
			Epoch:               big.NewInt(block.Certs[0].Epoch),
			ViewNumber:          big.NewInt(block.Certs[0].ViewNumber),
			Signature:           common.Hex2Bytes(block.Certs[0].Signature),
			ValidatorSignBitMap: big.NewInt(0),
			SignedCount:         big.NewInt(0),
		}
	}
	input, err := mapprotocol.Platon.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(ProofData{
		ReceiptProof: rp,
		Header:       block.Header,
		QuorumCert:   cert,
	})
	if err != nil {
		return nil, err
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}
	return pack, nil
}
