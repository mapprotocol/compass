package platon

import (
	"context"
	"math/big"

	"github.com/mapprotocol/compass/pkg/platon"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	iproof "github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
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
	Cert       *QuorumCert
}

func ConvertHeader(header *platon.Header) *BlockHeader {
	bloom := make([]byte, 0, len(header.Bloom))
	for _, b := range header.Bloom {
		bloom = append(bloom, b)
	}
	nonce := make([]byte, 0, len(header.Nonce))
	for _, b := range header.Nonce {
		nonce = append(nonce, b)
	}

	return &BlockHeader{
		ParentHash:       header.ParentHash.Bytes(),
		Miner:            common.HexToAddress(header.Coinbase.String()),
		StateRoot:        header.Root.Bytes(),
		TransactionsRoot: header.TxHash.Bytes(),
		ReceiptsRoot:     header.ReceiptHash.Bytes(),
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

type Validator struct {
	Address   common.Address
	NodeId    []byte
	BlsPubKey []byte
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

	input, err := mapprotocol.Platon.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(ProofData{
		ReceiptProof: rp,
		Header:       block.Header,
		QuorumCert:   block.Cert,
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

func GetHeaderParam(client *ethclient.Client, latestBlock *big.Int) (*UpdateBlock, error) {
	header, err := client.PlatonGetBlockByNumber(context.Background(), latestBlock)
	if err != nil {
		return nil, err
	}
	pHeader := ConvertHeader(header)
	validator, err := client.PlatonGetValidatorByNumber(context.Background(), new(big.Int).Add(pHeader.Number, big.NewInt(1)))
	if err != nil {
		return nil, err
	}
	quorumCert, err := client.PlatonGetBlockQuorumCertByHash(context.Background(), []common.Hash{common.HexToHash(header.Hash().String())})
	if err != nil {
		return nil, err
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
	if len(quorumCert) > 0 {
		vsb, sc := handlerValidatorSet(quorumCert[0].ValidatorSet)
		cert = &QuorumCert{
			BlockHash:           common.HexToHash(quorumCert[0].BlockHash),
			BlockIndex:          big.NewInt(quorumCert[0].BlockIndex),
			BlockNumber:         big.NewInt(quorumCert[0].BlockNumber),
			Epoch:               big.NewInt(quorumCert[0].Epoch),
			ViewNumber:          big.NewInt(quorumCert[0].ViewNumber),
			Signature:           common.Hex2Bytes(quorumCert[0].Signature),
			ValidatorSignBitMap: big.NewInt(vsb),
			SignedCount:         big.NewInt(sc),
		}
	}

	return &UpdateBlock{
		Header:     pHeader,
		Validators: validator,
		Cert:       cert,
	}, nil
}

func handlerValidatorSet(set string) (int64, int64) {
	var validatorSignBitMap, signedCount int64
	for idx, x := range set {
		if x != 'x' {
			continue
		}
		signedCount++
		validatorSignBitMap += int64(1 << (len(set) - idx - 1))
	}
	return validatorSignBitMap, signedCount
}
