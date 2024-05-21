package eth2

import (
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/proof"
	"github.com/mapprotocol/compass/pkg/util"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
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

func AssembleProof(header BlockHeader, log types.Log, receipts []*types.Receipt, method string, fId msg.ChainId, proofType int64) ([]byte, error) {
	txIndex := log.TxIndex
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
		pack, err = proof.Pack(fId, method, mapprotocol.Eth2, pd)
	case constant.ProofTypeOfZk:
	case constant.ProofTypeOfOracle:
		nr := mapprotocol.MapTxReceipt{
			PostStateOrStatus: receipt.PostStateOrStatus,
			CumulativeGasUsed: receipt.CumulativeGasUsed,
			Bloom:             receipt.Bloom,
			Logs:              receipt.Logs,
		}
		nrRlp, err := rlp.EncodeToBytes(nr)
		if err != nil {
			return nil, err
		}
		pd := proof.NewData{
			BlockNum: big.NewInt(int64(log.BlockNumber)),
			ReceiptProof: proof.NewReceiptProof{
				TxReceipt:   nrRlp,
				ReceiptType: receipt.ReceiptType,
				KeyIndex:    util.Key2Hex(key, len(prf)),
				Proof:       prf,
			},
		}
		pack, err = proof.Pack(fId, method, mapprotocol.ProofAbi, pd)
	}

	return pack, nil
}
