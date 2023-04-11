package eth2

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/mapprotocol/compass/internal/tx"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/ethclient"
	utils "github.com/mapprotocol/compass/shared/ethereum"
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

func GetProof(client *ethclient.Client, endPoint string, latestBlock *big.Int, log *types.Log, method string, fId msg.ChainId) ([]byte, error) {
	header, err := client.EthLatestHeaderByNumber(endPoint, latestBlock)
	if err != nil {
		return nil, err
	}
	// when syncToMap we need to assemble a tx proof
	txsHash, err := tx.GetTxsHashByBlockNumber(client, latestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to get tx hashes Logs: %w", err)
	}
	receipts, err := tx.GetReceiptsByTxsHash(client, txsHash)
	if err != nil {
		return nil, fmt.Errorf("unable to get receipts hashes Logs: %w", err)
	}
	return AssembleProof(*ConvertHeader(header), *log, receipts, method, fId)
}

func AssembleProof(header BlockHeader, log types.Log, receipts []*types.Receipt, method string, fId msg.ChainId) ([]byte, error) {
	txIndex := log.TxIndex
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])
	if err != nil {
		return nil, err
	}

	proof, err := getProof(receipts, txIndex)
	if err != nil {
		return nil, err
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))
	ek := utils.Key2Hex(key, len(proof))

	pd := ReceiptProof{
		Header:    header,
		TxReceipt: *receipt,
		KeyIndex:  ek,
		Proof:     proof,
	}

	input, err := mapprotocol.Eth2.Methods[mapprotocol.MethodOfGetBytes].Inputs.Pack(pd)
	if err != nil {
		return nil, err
	}

	//fmt.Println("bsc getBytes after hex ------------ ", "0x"+common.Bytes2Hex(input))
	pack, err := mapprotocol.PackInput(mapprotocol.Mcs, method, new(big.Int).SetUint64(uint64(fId)), input)
	//pack, err := mapprotocol.LightManger.Pack(mapprotocol.MethodVerifyProofData, new(big.Int).SetUint64(uint64(fId)), input)
	if err != nil {
		return nil, err
	}
	return pack, nil
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = utils.DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}

	return proof, nil
}
