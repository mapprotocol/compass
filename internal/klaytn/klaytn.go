package klaytn

import (
	"bytes"
	"context"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/util"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
)

type Header struct {
	ParentHash       []byte         `json:"parentHash"`
	Reward           common.Address `json:"reward"`
	StateRoot        []byte         `json:"stateRoot"`
	TransactionsRoot []byte         `json:"transactionsRoot"`
	ReceiptsRoot     []byte         `json:"receiptsRoot"`
	LogsBloom        []byte         `json:"logsBloom"`
	BlockScore       *big.Int       `json:"blockScore"`
	Number           *big.Int       `json:"number"`
	GasUsed          *big.Int       `json:"gasUsed"`
	Timestamp        *big.Int       `json:"timestamp"`
	TimestampFoS     *big.Int       `json:"timestampFoS"`
	ExtraData        []byte         `json:"extraData"`
	GovernanceData   []byte         `json:"governanceData"`
	VoteData         []byte         `json:"voteData"`
	BaseFee          *big.Int       `json:"baseFee"`
}

const (
	PrefixOfHex = "0x"
)

type RpcHeader struct {
	BaseFeePerGas    string         `json:"baseFeePerGas"`
	BlockScore       string         `json:"blockscore"`
	ExtraData        string         `json:"extraData"`
	GasUsed          string         `json:"gasUsed"`
	GovernanceData   string         `json:"governanceData"`
	Hash             common.Hash    `json:"hash"`
	LogsBloom        string         `json:"logsBloom"`
	Number           string         `json:"number"`
	ParentHash       common.Hash    `json:"parentHash"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	Reward           common.Address `json:"reward"`
	Size             string         `json:"size"`
	StateRoot        common.Hash    `json:"stateRoot"`
	Timestamp        string         `json:"timestamp"`
	TimestampFoS     string         `json:"timestampFoS"`
	TotalBlockScore  string         `json:"totalBlockScore"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	Transactions     []Transactions `json:"transactions"`
	VoteData         string         `json:"voteData"`
}

type Transactions struct {
	AccessList           []interface{} `json:"accessList,omitempty"`
	BlockHash            string        `json:"blockHash"`
	BlockNumber          string        `json:"blockNumber"`
	ChainID              string        `json:"chainId,omitempty"`
	From                 string        `json:"from"`
	Gas                  string        `json:"gas"`
	GasPrice             string        `json:"gasPrice"`
	Hash                 string        `json:"hash"`
	Input                string        `json:"input"`
	MaxFeePerGas         string        `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas string        `json:"maxPriorityFeePerGas,omitempty"`
	Nonce                string        `json:"nonce"`
	R                    string        `json:"r"`
	S                    string        `json:"s"`
	To                   string        `json:"to"`
	TransactionIndex     string        `json:"transactionIndex"`
	Type                 string        `json:"type"`
	V                    string        `json:"v"`
	Value                string        `json:"value"`
}

type GovernanceVote struct {
	Validator common.Address `json:"validator"`
	Key       string         `json:"key"`
	Value     interface{}    `json:"value"`
}

func ConvertContractHeader(ethHeader *types.Header, rh *RpcHeader) Header {
	bloom := make([]byte, 0, len(ethHeader.Bloom))
	for _, b := range ethHeader.Bloom {
		bloom = append(bloom, b)
	}
	blockScore := new(big.Int)
	blockScore.SetString(strings.TrimPrefix(rh.BlockScore, PrefixOfHex), 16)
	baseFeePerGas := new(big.Int)
	baseFeePerGas.SetString(strings.TrimPrefix(rh.BaseFeePerGas, PrefixOfHex), 16)
	timestamp := new(big.Int)
	timestamp.SetString(strings.TrimPrefix(rh.Timestamp, PrefixOfHex), 16)
	timestampFos := new(big.Int)
	timestampFos.SetString(strings.TrimPrefix(rh.TimestampFoS, PrefixOfHex), 16)
	return Header{
		ParentHash:       hashToByte(ethHeader.ParentHash),
		Reward:           rh.Reward,
		StateRoot:        hashToByte(ethHeader.Root),
		TransactionsRoot: hashToByte(ethHeader.TxHash),
		ReceiptsRoot:     hashToByte(ethHeader.ReceiptHash),
		LogsBloom:        bloom,
		BlockScore:       blockScore,
		BaseFee:          baseFeePerGas,
		Number:           ethHeader.Number,
		GasUsed:          new(big.Int).SetUint64(ethHeader.GasUsed),
		Timestamp:        timestamp,
		TimestampFoS:     timestampFos,
		ExtraData:        common.Hex2Bytes(strings.TrimPrefix(rh.ExtraData, PrefixOfHex)),
		GovernanceData:   common.Hex2Bytes(strings.TrimPrefix(rh.GovernanceData, PrefixOfHex)),
		VoteData:         common.Hex2Bytes(strings.TrimPrefix(rh.VoteData, PrefixOfHex)),
	}
}

func hashToByte(h common.Hash) []byte {
	ret := make([]byte, 0, len(h))
	for _, b := range h {
		ret = append(ret, b)
	}
	return ret
}

func GetTxsHashByBlockNumber(conn *Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.BlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txs = append(txs, common.HexToHash(tx.Hash))
	}
	return txs, nil
}

type ReceiptProofOriginal struct {
	Header    Header
	Proof     [][]byte
	TxReceipt []byte
	KeyIndex  []byte
}

type ReceiptProof struct {
	Proof     []byte
	DeriveSha DeriveShaOriginal
}

type DeriveShaOriginal uint8

const (
	DeriveShaOrigin DeriveShaOriginal = iota
	DeriveShaSimple
	DeriveShaConcat
)

type ReceiptRLP struct {
	Status  uint
	GasUsed uint64
	Bloom   types.Bloom
	Logs    []*types.Log
}

// ReceiptRlps implements DerivableList for receipts.
type ReceiptRlps []*ReceiptRLP

// Len returns the number of receipts in this list.
func (rs ReceiptRlps) Len() int { return len(rs) }

// EncodeIndex encodes the i'th receipt to w.
func (rs ReceiptRlps) EncodeIndex(i int, w *bytes.Buffer) {
	rlp.Encode(w, rs[i])
}

type TxLog struct {
	Addr   common.Address
	Topics [][]byte
	Data   []byte
}

func AssembleProof(cli *Client, header Header, log *types.Log, fId msg.ChainId, receipts []*types.Receipt, method string, proofType int64) ([]byte, error) {
	GetReceiptsByTxsHash(cli, receipts)
	receiptRlps := make(ReceiptRlps, 0, len(receipts))
	for _, receipt := range receipts {
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
		receiptRlps = append(receiptRlps, &ReceiptRLP{
			Status:  uint(receipt.Status),
			GasUsed: receipt.GasUsed,
			Bloom:   receipt.Bloom,
			Logs:    receipt.Logs,
		})
	}
	prf, err := proof.Get(receiptRlps, log.TxIndex)
	if err != nil {
		return nil, err
	}
	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(log.TxIndex))
	ek := util.Key2Hex(key, len(prf))
	receipt, err := GetTxReceipt(receipts[log.TxIndex])
	if err != nil {
		return nil, err
	}

	data, err := rlp.EncodeToBytes(receipt)
	if err != nil {
		return nil, err
	}

	pd := ReceiptProofOriginal{
		Header:    header,
		Proof:     prf,
		TxReceipt: data,
		KeyIndex:  ek,
	}

	input, err := mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, errors.Wrap(err, "getBytes pack")
	}
	finpd := ReceiptProof{
		Proof:     input,
		DeriveSha: DeriveShaOrigin,
	}
	input, err = mapprotocol.Klaytn.Methods[mapprotocol.MethodOfGetFinalBytes].Inputs.Pack(finpd)
	if err != nil {
		return nil, errors.Wrap(err, "getFinalBytes pack")
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}

	return pack, nil
}

func GetReceiptsByTxsHash(cli *Client, receipts []*types.Receipt) {
	for idx, receipt := range receipts {
		if receipt.Status != 0 {
			continue
		}
		kr, err := cli.TransactionReceiptRpcOutput(context.Background(), receipt.TxHash)
		if err != nil {
			return
		}
		txError, _ := big.NewInt(0).SetString(strings.TrimPrefix(kr["txError"].(string), "0x"), 16)
		receipts[idx].Status = txError.Uint64()
	}
}

type TxReceipt struct {
	PostStateOrStatus []byte
	CumulativeGasUsed *big.Int
	Bloom             []byte
	Logs              []mapprotocol.TxLog
}

func GetTxReceipt(receipt *types.Receipt) (*TxReceipt, error) {
	logs := make([]mapprotocol.TxLog, 0, len(receipt.Logs))
	for _, lg := range receipt.Logs {
		topics := make([][]byte, len(lg.Topics))
		for i := range lg.Topics {
			topics[i] = lg.Topics[i][:]
		}
		logs = append(logs, mapprotocol.TxLog{
			Addr:   lg.Address,
			Topics: topics,
			Data:   lg.Data,
		})
	}

	return &TxReceipt{
		PostStateOrStatus: mapprotocol.StatusEncoding(receipt),
		CumulativeGasUsed: new(big.Int).SetUint64(receipt.GasUsed),
		Bloom:             receipt.Bloom[:],
		Logs:              logs,
	}, nil
}
