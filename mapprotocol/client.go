package mapprotocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	log "github.com/ChainSafe/log15"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/mapprotocol/compass/pkg/util"
)

func GetMapTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.MAPBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		ele := common.HexToHash(tx.Hash)
		txs = append(txs, ele)
	}
	return txs, nil
}

func GetLastReceipt(conn *ethclient.Client, latestBlock *big.Int) (*types.Receipt, error) {
	query := eth.FilterQuery{
		FromBlock: latestBlock,
		ToBlock:   latestBlock,
	}
	lastLog, err := conn.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, err
	}
	receipt := maptypes.NewReceipt(nil, false, 0)
	rl := make([]*maptypes.Log, 0, len(lastLog))
	el := make([]*types.Log, 0, len(lastLog))
	for idx, ll := range lastLog {
		if idx == 0 {
			continue
		}
		if ll.TxHash != ll.BlockHash {
			continue
		}
		rl = append(rl, &maptypes.Log{
			Address:     ll.Address,
			Topics:      ll.Topics,
			Data:        ll.Data,
			BlockNumber: ll.BlockNumber,
			TxHash:      ll.TxHash,
			TxIndex:     ll.TxIndex,
			BlockHash:   ll.BlockHash,
			Index:       ll.Index,
			Removed:     ll.Removed,
		})
		tl := ll
		el = append(el, &tl)
	}
	receipt.Logs = rl
	receipt.Bloom = maptypes.CreateBloom(maptypes.Receipts{receipt})
	return &types.Receipt{
		Type:              receipt.Type,
		PostState:         receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             types.BytesToBloom(receipt.Bloom.Bytes()),
		Logs:              el,
		TxHash:            receipt.TxHash,
		ContractAddress:   receipt.ContractAddress,
		GasUsed:           receipt.GasUsed,
		BlockHash:         receipt.BlockHash,
		BlockNumber:       receipt.BlockNumber,
		TransactionIndex:  receipt.TransactionIndex,
	}, nil
}

type Zk struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Height string `json:"height"`
		Status int    `json:"status"`
		Result struct {
			Proof struct {
				PiA      []string   `json:"pi_a"`
				PiB      [][]string `json:"pi_b"`
				PiC      []string   `json:"pi_c"`
				Protocol string     `json:"protocol"`
			} `json:"proof"`
			PublicInput []string `json:"public_input"`
		} `json:"result"`
		ErrorMsg string `json:"error_msg"`
	} `json:"data"`
}

func GetZkProof(endpoint string, cid msg.ChainId, height uint64) ([]*big.Int, error) {
	ret := make([]*big.Int, 0, 8)
	for {
		resp, err := http.Get(fmt.Sprintf("%s/proof?chain_id=%d&height=%d", endpoint, cid, height))
		if err != nil {
			util.Alarm(context.Background(), fmt.Sprintf("GetZkProof cid(%d) request failed, err is %v", height, err))
			log.Error("GetZkProof request failed", "err", err, "height", height, "cid", cid)
			time.Sleep(constant.BlockRetryInterval)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("GetZkProof read body failed", "err", err)
			time.Sleep(constant.BlockRetryInterval)
			continue
		}
		_ = resp.Body.Close()
		zk := &Zk{}
		err = json.Unmarshal(body, zk)
		if err != nil {
			util.Alarm(context.Background(), fmt.Sprintf("GetZkProof cid(%d) Unmarshal failed, err is %v", height, err))
			log.Error("GetZkProof Unmarshal failed", "err", err, "data", string(body))
			time.Sleep(constant.BlockRetryInterval)
			continue
		}
		// check status
		if zk.Data.Status != 3 {
			//util.Alarm(context.Background(), fmt.Sprintf("GetZkProof cid(%d) height(%d) Proof Not Ready", cid, height))
			log.Info("GetZkProof Proof Not Read", "cid", cid, "height", height)
			time.Sleep(constant.BalanceRetryInterval)
			continue
		}
		ret = append(ret, getId(zk.Data.Result.Proof.PiA)...)
		for _, bs := range zk.Data.Result.Proof.PiB {
			ret = append(ret, getId(bs)...)
		}
		ret = append(ret, getId(zk.Data.Result.Proof.PiC)...)
		break
	}
	return ret, nil
}

func GetCurValidators(cli *ethclient.Client, number *big.Int) ([]byte, error) {
	snapshot, err := cli.GetValidatorsBLSPublicKeys(context.Background(), number)
	if err != nil {
		return nil, err
	}

	ret := make([][]byte, 0)
	for _, v := range snapshot {
		ele := make([]byte, 0)
		for _, k := range v {
			ele = append(ele, k)
		}
		ret = append(ret, ele)
	}

	return makeValidatorInfo(ret), nil
}

func getId(ss []string) []*big.Int {
	ret := make([]*big.Int, 0, len(ss))
	for _, s := range ss {
		if s == "0" || s == "1" || len(s) <= 1 {
			continue
		}
		setString, ok := big.NewInt(0).SetString(s, 10)
		if !ok {
			continue
		}
		ret = append(ret, setString)
	}
	return ret
}

var (
	PUBLENGTH = 128
)

func makeValidatorInfo(blsPubkeys [][]byte) []byte {
	data := make([]byte, 0)
	count := len(blsPubkeys)
	left := 0
	if PUBLENGTH > count {
		left = PUBLENGTH - count
	}
	length := left * 160
	data1 := make([]byte, length)
	// info: pubkey+weight
	for i := 0; i < count; i++ {
		data = append(data, blsPubkeys[i]...)
		weight := []byte{0x01}
		weight = padTo32Bytes(weight)
		data = append(data, weight...)
	}
	data = append(data, data1...)
	return data
}

func padTo32Bytes(data []byte) []byte {
	paddingSize := 32 - len(data)
	if paddingSize <= 0 {
		return data
	}
	paddedData := make([]byte, 32)
	copy(paddedData[paddingSize:], data)
	return paddedData
}
