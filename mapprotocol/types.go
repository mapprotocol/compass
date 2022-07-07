package mapprotocol

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/atlas/consensus/istanbul/validator"
	"github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/atlas/helper/bls"
	"github.com/mapprotocol/compass/pkg/ethclient"

	"github.com/ethereum/go-ethereum/common"
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

func GetAggPK(cli *ethclient.Client, number *big.Int, extra []byte) (*G2, error) {
	var istanbulExtra *types.IstanbulExtra
	if err := rlp.DecodeBytes(extra[32:], &istanbulExtra); err != nil {
		return nil, err
	}

	snapshot, err := cli.GetSnapshot(context.Background(), number)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		pks = append(pks, pk)
	}

	aggPKBytes := bls.AggregatePK(pks).Marshal()
	return &G2{
		Xr: new(big.Int).SetBytes(aggPKBytes[32:64]),
		Xi: new(big.Int).SetBytes(aggPKBytes[:32]),
		Yr: new(big.Int).SetBytes(aggPKBytes[96:128]),
		Yi: new(big.Int).SetBytes(aggPKBytes[64:96]),
	}, nil
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
	MixDigest   common.Hash      `json:"minDigest"`
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
