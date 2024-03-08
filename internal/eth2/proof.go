package eth2

import (
	"github.com/mapprotocol/compass/internal/mapo"
	"github.com/mapprotocol/compass/internal/proof"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	log.Debug("eth exec", "path", execPath, "cmd", c.String())
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

func AssembleProof(header BlockHeader, log types.Log, receipts []*types.Receipt, method string, fId msg.ChainId) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	prf, err := proof.Get(types.Receipts(receipts), txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := mapo.Key2Hex(key, len(prf))

	pd := ReceiptProof{
		Header:    header,
		TxReceipt: *receipt,
		KeyIndex:  ek,
		Proof:     prf,
	}

	input, err := mapprotocol.Eth2.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}

	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}
	return pack, nil
}
