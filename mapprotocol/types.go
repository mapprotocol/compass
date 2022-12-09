package mapprotocol

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/atlas/consensus/istanbul/validator"
	"github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/atlas/helper/bls"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var (
	receiptStatusFailedRLP     = []byte{}
	receiptStatusSuccessfulRLP = []byte{0x01}
)

type G2 struct {
	Xr *big.Int
	Xi *big.Int
	Yr *big.Int
	Yi *big.Int
}

type BlockHeader struct {
	ParentHash  []byte
	Coinbase    common.Address
	Root        []byte
	TxHash      []byte
	ReceiptHash []byte
	Bloom       []byte
	Number      *big.Int
	GasLimit    *big.Int
	GasUsed     *big.Int
	Time        *big.Int
	ExtraData   []byte
	MixDigest   []byte
	Nonce       []byte
	BaseFee     *big.Int
}

type TxLog struct {
	Addr   common.Address
	Topics [][]byte
	Data   []byte
}

type TxReceipt struct {
	ReceiptType       *big.Int
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             []byte
	Logs              []TxLog
}

type ReceiptProof struct {
	Header   *BlockHeader
	AggPk    *G2
	Receipt  *TxReceipt
	KeyIndex []byte
	Proof    [][]byte
}

type NewMapReceiptProof struct {
	Header       *BlockHeader
	AggPk        *G2
	KeyIndex     []byte
	Proof        [][]byte
	Ist          IstanbulExtra
	TxReceiptRlp TxReceiptRlp
}

type IstanbulExtra struct {
	// Validators are the validators that have been added in the block
	Validators []common.Address
	// AddedPubKey are the BLS public keys for the validators added in the block
	AddedPubKey [][]byte
	// AddedG1PubKey are the BLS public keys for the validators added in the block
	AddedG1PubKey [][]byte
	// RemoveList is a bitmap having an active bit for each removed validator in the block
	RemoveList *big.Int
	// Seal is an ECDSA signature by the proposer
	Seal []byte
	// AggregatedSeal contains the aggregated BLS signature created via IBFT consensus.
	AggregatedSeal IstanbulAggregatedSeal
	// ParentAggregatedSeal contains and aggregated BLS signature for the previous block.
	ParentAggregatedSeal IstanbulAggregatedSeal
}

type IstanbulAggregatedSeal struct {
	Bitmap    *big.Int
	Signature []byte
	Round     *big.Int
}

func ConvertIstanbulExtra(istanbulExtra *types.IstanbulExtra) *IstanbulExtra {
	addedPubKey := make([][]byte, 0, len(istanbulExtra.AddedValidatorsPublicKeys))
	for _, avpk := range istanbulExtra.AddedValidatorsPublicKeys {
		data := make([]byte, 0, len(avpk))
		for _, v := range avpk {
			data = append(data, v)
		}
		addedPubKey = append(addedPubKey, data)
	}
	addedValidatorsG1PublicKeys := make([][]byte, 0, len(istanbulExtra.AddedValidatorsG1PublicKeys))
	for _, avgpk := range istanbulExtra.AddedValidatorsG1PublicKeys {
		data := make([]byte, 0, len(avgpk))
		for _, v := range avgpk {
			data = append(data, v)
		}
		addedValidatorsG1PublicKeys = append(addedPubKey, data)
	}

	return &IstanbulExtra{
		Validators:    istanbulExtra.AddedValidators,
		AddedPubKey:   addedPubKey,
		AddedG1PubKey: addedValidatorsG1PublicKeys,
		RemoveList:    istanbulExtra.RemovedValidators,
		Seal:          istanbulExtra.Seal,
		AggregatedSeal: IstanbulAggregatedSeal{
			Bitmap:    istanbulExtra.AggregatedSeal.Bitmap,
			Signature: istanbulExtra.AggregatedSeal.Signature,
			Round:     istanbulExtra.AggregatedSeal.Round,
		},
		ParentAggregatedSeal: IstanbulAggregatedSeal{
			Bitmap:    istanbulExtra.ParentAggregatedSeal.Bitmap,
			Signature: istanbulExtra.ParentAggregatedSeal.Signature,
			Round:     istanbulExtra.ParentAggregatedSeal.Round,
		},
	}
}

type TxReceiptRlp struct {
	ReceiptType *big.Int
	ReceiptRlp  []byte
}

type MapTxReceipt struct {
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             []byte
	Logs              []TxLog
}

type NewReceiptProof struct {
	Router   common.Address
	Coin     common.Address
	SrcChain *big.Int
	DstChain *big.Int
	TxProve  []byte
}

type TxProve struct {
	Receipt     *ethtypes.Receipt
	Prove       light.NodeList
	BlockNumber uint64
	TxIndex     uint
}

func ConvertHeader(header *types.Header) *BlockHeader {
	h := &BlockHeader{
		ParentHash:  header.ParentHash[:],
		Coinbase:    header.Coinbase,
		Root:        header.Root[:],
		TxHash:      header.TxHash[:],
		ReceiptHash: header.ReceiptHash[:],
		Bloom:       header.Bloom[:],
		Number:      header.Number,
		GasLimit:    new(big.Int).SetUint64(header.GasLimit),
		GasUsed:     new(big.Int).SetUint64(header.GasUsed),
		Time:        new(big.Int).SetUint64(header.Time),
		ExtraData:   header.Extra,
		MixDigest:   header.MixDigest[:],
		Nonce:       header.Nonce[:],
		BaseFee:     header.BaseFee,
	}
	return h
}

func GetAggPK(cli *ethclient.Client, number *big.Int, extra []byte) (*G2, *types.IstanbulExtra, []byte, error) {
	var istanbulExtra *types.IstanbulExtra
	if err := rlp.DecodeBytes(extra[32:], &istanbulExtra); err != nil {
		return nil, nil, nil, err
	}

	snapshot, err := cli.GetSnapshot(context.Background(), number)
	if err != nil {
		return nil, nil, nil, err
	}

	validators := validator.MapValidatorsToDataWithBLSKeyCache(snapshot.ValSet.List())
	publicKeys := make([]bls.SerializedPublicKey, 0)
	for i, v := range validators {
		if istanbulExtra.AggregatedSeal.Bitmap.Bit(i) == 1 {
			publicKeys = append(publicKeys, v.BLSPublicKey)
		}
	}

	var pks []*bls.PublicKey
	for _, v := range publicKeys {
		pk, err := bls.UnmarshalPk(v[:])
		if err != nil {
			return nil, nil, nil, err
		}
		pks = append(pks, pk)
	}

	aggPKBytes := bls.AggregatePK(pks).Marshal()
	return &G2{
		Xi: new(big.Int).SetBytes(aggPKBytes[:32]),
		Xr: new(big.Int).SetBytes(aggPKBytes[32:64]),
		Yi: new(big.Int).SetBytes(aggPKBytes[64:96]),
		Yr: new(big.Int).SetBytes(aggPKBytes[96:128]),
	}, istanbulExtra, aggPKBytes, nil
}

func GetTxReceipt(receipt *ethtypes.Receipt) (*TxReceipt, error) {
	logs := make([]TxLog, 0, len(receipt.Logs))
	for _, lg := range receipt.Logs {
		topics := make([][]byte, len(lg.Topics))
		for i := range lg.Topics {
			topics[i] = lg.Topics[i][:]
		}
		logs = append(logs, TxLog{
			Addr:   lg.Address,
			Topics: topics,
			Data:   lg.Data,
		})
	}

	return &TxReceipt{
		ReceiptType:       new(big.Int).SetUint64(uint64(receipt.Type)),
		PostStateOrStatus: StatusEncoding(receipt),
		CumulativeGasUsed: new(big.Int).SetUint64(receipt.CumulativeGasUsed),
		Bloom:             receipt.Bloom[:],
		Logs:              logs,
	}, nil
}

func StatusEncoding(r *ethtypes.Receipt) []byte {
	if len(r.PostState) == 0 {
		if r.Status == types.ReceiptStatusFailed {
			return receiptStatusFailedRLP
		}
		return receiptStatusSuccessfulRLP
	}
	return r.PostState
}

type NearNeedHeader struct {
	ParentHash  common.Hash      `json:"parentHash"       gencodec:"required"`
	Coinbase    common.Address   `json:"coinbase"            gencodec:"required"`
	Root        common.Hash      `json:"root"         gencodec:"required"`
	TxHash      common.Hash      `json:"txHash" gencodec:"required"`
	ReceiptHash common.Hash      `json:"receiptHash"     gencodec:"required"`
	Bloom       types.Bloom      `json:"bloom"        gencodec:"required"`
	Number      *hexutil.Big     `json:"number"           gencodec:"required"`
	GasLimit    hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
	GasUsed     hexutil.Uint64   `json:"gasUsed"          gencodec:"required"`
	Time        hexutil.Uint64   `json:"time"        gencodec:"required"`
	Extra       hexutil.Bytes    `json:"extra"        gencodec:"required"`
	MixDigest   common.Hash      `json:"mixDigest"`
	Nonce       types.BlockNonce `json:"nonce"`
	BaseFee     *hexutil.Big     `json:"baseFee" rlp:"optional"`
	Hash        common.Hash      `json:"hash"`
}

func ConvertNearNeedHeader(h *types.Header) *NearNeedHeader {
	var enc NearNeedHeader
	enc.ParentHash = h.ParentHash
	enc.Coinbase = h.Coinbase
	enc.Root = h.Root
	enc.TxHash = h.TxHash
	enc.ReceiptHash = h.ReceiptHash
	enc.Bloom = h.Bloom
	enc.Number = (*hexutil.Big)(h.Number)
	enc.GasLimit = hexutil.Uint64(h.GasLimit)
	enc.GasUsed = hexutil.Uint64(h.GasUsed)
	enc.Time = hexutil.Uint64(h.Time)
	enc.Extra = h.Extra
	enc.MixDigest = h.MixDigest
	enc.Nonce = h.Nonce
	enc.BaseFee = (*hexutil.Big)(h.BaseFee)
	enc.Hash = h.Hash()
	return &enc
}

type NearReceiptProof struct {
	BlockHeaderLite  BlockHeaderLite    `json:"block_header_lite"`
	BlockProof       []BlockProof       `json:"block_proof"`
	OutcomeProof     OutcomeProof       `json:"outcome_proof"`
	OutcomeRootProof []OutcomeRootProof `json:"outcome_root_proof"`
}

type InnerLite struct {
	BlockMerkleRoot  string `json:"block_merkle_root"`
	EpochID          string `json:"epoch_id"`
	Height           int    `json:"height"`
	NextBpHash       string `json:"next_bp_hash"`
	NextEpochID      string `json:"next_epoch_id"`
	OutcomeRoot      string `json:"outcome_root"`
	PrevStateRoot    string `json:"prev_state_root"`
	Timestamp        int64  `json:"timestamp"`
	TimestampNanosec string `json:"timestamp_nanosec"`
}

type BlockHeaderLite struct {
	InnerLite     InnerLite `json:"inner_lite"`
	InnerRestHash string    `json:"inner_rest_hash"`
	PrevBlockHash string    `json:"prev_block_hash"`
}

type BlockProof struct {
	Direction string `json:"direction"`
	Hash      string `json:"hash"`
}

type GasProfile struct {
	Cost         string `json:"cost"`
	CostCategory string `json:"cost_category"`
	GasUsed      string `json:"gas_used"`
}

type Metadata struct {
	GasProfile []GasProfile `json:"gas_profile"`
	Version    int          `json:"version"`
}

type Status struct {
	SuccessValue string `json:"SuccessValue"`
}

type Outcome struct {
	ExecutorID  string        `json:"executor_id"`
	GasBurnt    int64         `json:"gas_burnt"`
	Logs        []interface{} `json:"logs"`
	Metadata    Metadata      `json:"metadata"`
	ReceiptIds  []string      `json:"receipt_ids"`
	Status      Status        `json:"status"`
	TokensBurnt string        `json:"tokens_burnt"`
}

type Proof struct {
	Direction string `json:"direction"`
	Hash      string `json:"hash"`
}

type OutcomeProof struct {
	BlockHash string  `json:"block_hash"`
	ID        string  `json:"id"`
	Outcome   Outcome `json:"outcome"`
	Proof     []Proof `json:"proof"`
}

type OutcomeRootProof struct {
	Direction string `json:"direction"`
	Hash      string `json:"hash"`
}
