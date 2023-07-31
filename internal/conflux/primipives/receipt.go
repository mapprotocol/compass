package primitives

import (
	"fmt"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/pkg/errors"
)

type Receipt struct {
	AccumulatedGasUsed    *big.Int
	GasFee                *big.Int
	GasSponsorPaid        Bool
	LogBloom              []byte
	Logs                  []TxLog
	OutcomeStatus         uint8
	StorageSponsorPaid    Bool
	StorageCollateralized []StorageChange
	StorageReleased       []StorageChange
}

func MustRLPEncodeReceipt(receipt *types.TransactionReceipt) []byte {
	val := ConvertReceipt(receipt)
	encoded, err := rlp.EncodeToBytes(val)
	if err != nil {
		panic(err)
	}
	return encoded
}

func ConvertReceipt(receipt *types.TransactionReceipt) Receipt {
	storageCollateralized, storageReleased := constructStorageChanges(receipt)

	for _, log := range receipt.Logs {
		fmt.Println("log.BlockHash", log.BlockHash)
		fmt.Println("log.Data", "0x"+common.Bytes2Hex(log.Data))
		fmt.Println("log.BlockNumber", log.BlockHash)
		fmt.Println("log.EpochNumber", log.EpochNumber)
		fmt.Println("log.Address", log.Address)
		fmt.Println("log.TransactionHash", log.TransactionHash)
		fmt.Println("log.TransactionIndex", log.TransactionIndex)
		fmt.Println("log.LogIndex", log.LogIndex)
		fmt.Println("log.Space", log.Space)
		fmt.Println("log.Topics", log.Topics)
	}

	return Receipt{
		AccumulatedGasUsed:    receipt.AccumulatedGasUsed.ToInt(),
		GasFee:                receipt.GasFee.ToInt(),
		GasSponsorPaid:        Bool(receipt.GasCoveredBySponsor),
		LogBloom:              hexutil.MustDecode(string(receipt.LogsBloom)),
		Logs:                  convertLogs(receipt.Logs),
		OutcomeStatus:         uint8(receipt.MustGetOutcomeType()),
		StorageSponsorPaid:    Bool(receipt.StorageCoveredBySponsor),
		StorageCollateralized: storageCollateralized,
		StorageReleased:       storageReleased,
	}
}

type StorageChange struct {
	Account     common.Address
	Collaterals uint64
}

func constructStorageChanges(receipt *types.TransactionReceipt) (collateralized, released []StorageChange) {
	for _, v := range receipt.StorageReleased {
		released = append(released, StorageChange{
			Account:     v.Address.MustGetCommonAddress(),
			Collaterals: uint64(v.Collaterals),
		})
	}

	if receipt.StorageCollateralized == 0 {
		return
	}

	var account types.Address
	if receipt.StorageCoveredBySponsor {
		account = *receipt.To
	} else {
		account = receipt.From
	}

	collateralized = append(collateralized, StorageChange{
		Account:     account.MustGetCommonAddress(),
		Collaterals: uint64(receipt.StorageCollateralized),
	})

	return
}

const (
	LogSpaceNative   uint8 = 1
	LogSpaceEthereum uint8 = 2
)

type TxLog struct {
	Addr   common.Address
	Topics []common.Hash
	Data   []byte
	Space  uint8
}

// EncodeRLP implements the rlp.Encoder interface.
func (log TxLog) EncodeRLP(w io.Writer) error {
	switch log.Space {
	case LogSpaceNative:
		return rlp.Encode(w, []interface{}{log.Addr, log.Topics, log.Data})
	case LogSpaceEthereum:
		return rlp.Encode(w, []interface{}{log.Addr, log.Topics, log.Data, log.Space})
	default:
		return errors.Errorf("invalid log space %v", log.Space)
	}
}

func convertLogs(logs []types.Log) []TxLog {
	var result []TxLog

	for _, v := range logs {
		var topics []common.Hash
		for _, t := range v.Topics {
			topics = append(topics, *t.ToCommonHash())
		}

		var space uint8
		switch *v.Space {
		case types.SPACE_NATIVE:
			space = LogSpaceNative
		case types.SPACE_EVM:
			space = LogSpaceEthereum
		default:
			panic("invalid space in log entry")
		}

		result = append(result, TxLog{
			Addr:   v.Address.MustGetCommonAddress(),
			Topics: topics,
			Data:   v.Data,
			Space:  space,
		})
	}

	return result
}
