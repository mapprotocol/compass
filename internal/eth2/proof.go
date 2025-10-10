package eth2

import (
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/pkg/errors"
)

var execPath = "./eth2-proof"

func init() {
	if filepath.Dir(os.Args[0]) == "." {
		return
	}
	execPath = filepath.Join(filepath.Dir(os.Args[0]), "eth2-proof")
}

func Generate(slot, endpoint string) ([][32]byte, string, string, error) {
	c := exec.Command(execPath, "generate", "--slot", slot, "--endpoint", endpoint)
	log.Info("eth exec", "path", execPath, "cmd", c.String())
	subOutPut, err := c.CombinedOutput()
	if err != nil {
		return nil, "", "", errors.Wrap(err, "command exec failed")
	}

	outPuts := strings.Split(string(subOutPut), "\n")
	ret := make([][32]byte, 0, len(outPuts))
	var txRoot, wdRoot string
	for _, op := range outPuts {
		if strings.HasPrefix(op, "0x") {
			ret = append(ret, common.HexToHash(op))
		} else if strings.HasPrefix(op, "txRoot") {
			txRoot = strings.TrimSpace(strings.TrimPrefix(op, "txRoot"))
		} else if strings.HasPrefix(op, "wdRoot") {
			wdRoot = strings.TrimSpace(strings.TrimPrefix(op, "wdRoot"))
		}
	}

	return ret, txRoot, wdRoot, nil
}

func GenerateByApi(slot []string) [][32]byte {
	ret := make([][32]byte, 0, len(slot))
	for _, op := range slot {
		ret = append(ret, common.HexToHash(op))
	}

	return ret
}

type ReceiptProof struct {
	Header    BlockHeader
	TxReceipt mapprotocol.TxReceipt
	KeyIndex  []byte
	Proof     [][]byte
}

func AssembleProof(header BlockHeader, log *types.Log, receipts []*types.Receipt, method string, fId msg.ChainId,
	proofType int64, sign [][]byte) ([]byte, error) {
	txIndex := log.TxIndex
	orderId := log.Topics[1]
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}
	pr := Receipts{}
	for _, r := range receipts {
		pr = append(pr, &Receipt{Receipt: r})
	}
	prf, err := proof.Get(pr, txIndex)
	if err != nil {
		return nil, err
	}
	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))

	idx := 0
	for i, ele := range receipts[txIndex].Logs {
		if ele.Index != log.Index {
			continue
		}
		idx = i
	}

	var pack []byte
	switch proofType {
	case constant.ProofTypeOfOrigin:
		ek := mapo.Key2Hex(key, len(prf))
		pd := ReceiptProof{
			Header:    header,
			TxReceipt: *receipt,
			KeyIndex:  ek,
			Proof:     prf,
		}
		pack, err = proof.V3Pack(fId, method, mapprotocol.Eth2, idx, orderId, false, pd)
	case constant.ProofTypeOfZk:
	case constant.ProofTypeOfOracle:
		pack, err = proof.Oracle(header.Number.Uint64(), receipt, key, prf, fId, method, idx,
			mapprotocol.ProofAbi, orderId, false)
	case constant.ProofTypeOfNewOracle:
		fallthrough
	case constant.ProofTypeOfLogOracle:
		pack, err = proof.SignOracle(&maptypes.Header{
			ReceiptHash: header.ReceiptsRoot,
			Number:      big.NewInt(int64(log.BlockNumber)),
		}, receipt, key, prf, fId, idx, method, sign, log, proofType)
	}

	if err != nil {
		return nil, err
	}
	return pack, nil
}
