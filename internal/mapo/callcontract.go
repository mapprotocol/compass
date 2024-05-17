// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapo

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"strings"

	"github.com/mapprotocol/compass/internal/arb"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/op"
	"github.com/mapprotocol/compass/internal/scroll"
	"github.com/mapprotocol/compass/pkg/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	"github.com/pkg/errors"
)

func AssembleEthProof(conn *ethclient.Client, log *types.Log, receipts []*types.Receipt, method string, fId msg.ChainId, proofType int64) ([]byte, error) {
	var (
		pack []byte
		err  error
	)
	switch proofType {
	case constant.ProofTypeOfOrigin:
	case constant.ProofTypeOfZk:
	case constant.ProofTypeOfOracle:
		receipt, err := mapprotocol.GetTxReceipt(receipts[log.TxIndex])
		if err != nil {
			return nil, err
		}

		prf, err := ethProof(conn, fId, log.TxIndex, receipts)
		if err != nil {
			return nil, err
		}

		var key []byte
		key = rlp.AppendUint64(key[:0], uint64(log.TxIndex))
		pack, err = proof.Oracle(log.BlockNumber, receipt, key, prf, fId, method, log.TxIndex, mapprotocol.ProofAbi)
	}

	if err != nil {
		return nil, err
	}

	return pack, nil
}

func ethProof(conn *ethclient.Client, fId msg.ChainId, txIdx uint, receipts []*types.Receipt) ([][]byte, error) {
	var dls proof.DerivableList
	switch fId {
	case constant.ArbChainId, constant.ArbTestnetChainId, constant.MantleChainId:
		pr := arb.Receipts{}
		for _, r := range receipts {
			pr = append(pr, &arb.Receipt{Receipt: r})
		}
		dls = pr
	case constant.ScrollChainId:
		pr := scroll.Receipts{}
		for _, r := range receipts {
			pr = append(pr, &scroll.Receipt{Receipt: r})
		}
		dls = pr
	case constant.OpChainId, constant.BaseChainId, constant.BlastChainId:
		pr := op.Receipts{}
		for _, r := range receipts {
			tmp, err := conn.OpReceipt(context.Background(), r.TxHash)
			if err != nil {
				continue
			}
			vptr := uint64(0)
			nptr := uint64(0)
			if tmp.DepositReceiptVersion != "" {
				version, _ := big.NewInt(0).SetString(strings.TrimPrefix(tmp.DepositReceiptVersion, "0x"), 16)
				vptr = version.Uint64()
			}
			if tmp.DepositNonce != "" {
				nonce, _ := big.NewInt(0).SetString(strings.TrimPrefix(tmp.DepositNonce, "0x"), 16)
				nptr = nonce.Uint64()
			}
			pr = append(pr, &op.Receipt{Receipt: r, DepositReceiptVersion: &vptr, DepositNonce: &nptr})
		}
		dls = pr
	default:
		dls = types.Receipts(receipts)
	}
	ret, err := proof.Get(dls, txIdx)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func AssembleMapProof(cli *ethclient.Client, log *types.Log, receipts []*types.Receipt,
	header *maptypes.Header, fId msg.ChainId, method, zkUrl string, proofType int64) (uint64, []byte, error) {
	toChainID := log.Topics[2]
	uToChainID := binary.BigEndian.Uint64(toChainID[len(toChainID)-8:])
	txIndex := log.TxIndex
	aggPK, ist, aggPKBytes, err := mapprotocol.GetAggPK(cli, new(big.Int).Sub(header.Number, big.NewInt(1)), header.Extra)
	if err != nil {
		return 0, nil, err
	}

	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	prf, err := proof.Get(types.Receipts(receipts), txIndex)
	if err != nil {
		return 0, nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := util.Key2Hex(key, len(prf))
	if name, ok := mapprotocol.OnlineChaId[msg.ChainId(uToChainID)]; ok && strings.ToLower(name) != "near" {
		istanbulExtra := mapprotocol.ConvertIstanbulExtra(ist)
		nr := mapprotocol.MapTxReceipt{
			PostStateOrStatus: receipt.PostStateOrStatus,
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom,
			Logs:              receipt.Logs,
		}

		nrRlp, err := rlp.EncodeToBytes(nr)
		if err != nil {
			return 0, nil, err
		}
		rp := mapprotocol.NewMapReceiptProof{
			Header:   mapprotocol.ConvertHeader(header),
			AggPk:    aggPK,
			KeyIndex: ek,
			Proof:    prf,
			Ist:      *istanbulExtra,
			TxReceiptRlp: mapprotocol.TxReceiptRlp{
				ReceiptType: receipt.ReceiptType,
				ReceiptRlp:  nrRlp,
			},
		}

		if (uToChainID == 71 || uToChainID == 1030) && method == mapprotocol.MethodOfSwapIn {
			method = mapprotocol.MethodOfSwapInWithIndex
		}

		var payloads []byte
		switch proofType {
		case constant.ProofTypeOfOrigin:
			payloads, err = proof.V3Pack(fId, method, mapprotocol.Map2Other, log.TxIndex, rp)
		case constant.ProofTypeOfZk:
			zkProof, err := mapprotocol.GetZkProof(zkUrl, fId, header.Number.Uint64())
			if err != nil {
				return 0, nil, errors.Wrap(err, "GetZkProof failed")
			}
			payloads, err = proof.Pack(fId, method, mapprotocol.Mcs, rp, zkProof)
		case constant.ProofTypeOfOracle:
			if uToChainID == constant.EthChainId {
				pd := proof.Data{
					BlockNum: header.Number,
					ReceiptProof: proof.ReceiptProof{
						TxReceipt: *receipt,
						KeyIndex:  util.Key2Hex(key, len(prf)),
						Proof:     prf,
					},
				}

				payloads, err = proof.Pack(fId, method, mapprotocol.OracleAbi, pd)
			} else {
				payloads, err = proof.Oracle(header.Number.Uint64(), receipt, key, prf, fId, method, log.TxIndex, mapprotocol.ProofAbi)
			}
		}

		if err != nil {
			return 0, nil, err
		}
		return uToChainID, payloads, nil
	}

	bytesBuffer := bytes.NewBuffer([]byte{})
	err = binary.Write(bytesBuffer, binary.LittleEndian, uint64(txIndex))
	if err != nil {
		return 0, nil, err
	}

	nProof := make([]string, 0, len(prf))
	for _, p := range prf {
		nProof = append(nProof, "0x"+common.Bytes2Hex(p))
	}
	m := map[string]interface{}{
		"header": mapprotocol.ConvertNearNeedHeader(header),
		"agg_pk": map[string]interface{}{
			"xr": "0x" + common.Bytes2Hex(aggPKBytes[32:64]),
			"xi": "0x" + common.Bytes2Hex(aggPKBytes[:32]),
			"yi": "0x" + common.Bytes2Hex(aggPKBytes[64:96]),
			"yr": "0x" + common.Bytes2Hex(aggPKBytes[96:128]),
		},
		"key_index": "0x" + common.Bytes2Hex(key),
		"receipt":   ConvertNearReceipt(receipt),
		"proof":     nProof,
	}

	idx := 0
	match := false
	for lIdx, l := range receipt.Logs {
		for _, topic := range l.Topics {
			if common.BytesToHash(topic) == log.Topics[0] {
				idx = lIdx
				match = true
				break
			}
		}
		if match {
			break
		}
	}
	data, _ := json.Marshal(map[string]interface{}{
		"receipt_proof": m,
		"index":         idx,
	})
	return uToChainID, data, nil
}

func Key2Hex(str []byte, proofLength int) []byte {
	ret := make([]byte, 0)
	if len(ret)+1 == proofLength {
		ret = append(ret, str...)
	} else {
		for _, b := range str {
			ret = append(ret, b/16)
			ret = append(ret, b%16)
		}
	}
	return ret
}

type TxReceipt struct {
	ReceiptType       string  `json:"receipt_type"`
	PostStateOrStatus string  `json:"post_state_or_status"`
	CumulativeGasUsed string  `json:"cumulative_gas_used"`
	Bloom             string  `json:"bloom"`
	Logs              []TxLog `json:"logs"`
}

type TxLog struct {
	Address common.Address `json:"address"`
	Topics  []string       `json:"topics"`
	Data    string         `json:"data"`
}

func ConvertNearReceipt(h *mapprotocol.TxReceipt) *TxReceipt {
	logs := make([]TxLog, 0, len(h.Logs))
	for _, log := range h.Logs {
		topics := make([]string, 0, len(log.Topics))
		for _, t := range log.Topics {
			topics = append(topics, "0x"+common.Bytes2Hex(t))
		}
		logs = append(logs, TxLog{
			Address: log.Addr,
			Topics:  topics,
			Data:    "0x" + common.Bytes2Hex(log.Data),
		})
	}
	return &TxReceipt{
		ReceiptType:       h.ReceiptType.String(),
		PostStateOrStatus: "0x" + common.Bytes2Hex(h.PostStateOrStatus),
		CumulativeGasUsed: h.CumulativeGasUsed.String(),
		Bloom:             "0x" + common.Bytes2Hex(h.Bloom),
		Logs:              logs,
	}
}
