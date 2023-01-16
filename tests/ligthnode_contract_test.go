package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mapprotocol/compass/internal/eth2"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mapprotocol/compass/pkg/redis"

	"github.com/mapprotocol/compass/internal/near"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/ethereum/go-ethereum"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/ethclient"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

var (
	err        error
	eth2Client *eth2.Client
)

func TestClient_BeaconHeaders(t *testing.T) {
	t.Log("------------- ")
	//eth2Client, err = eth2.DialHttp("https://lodestar-mainnet.chainsafe.io")
	eth2Client, err = eth2.DialHttp("http://18.138.248.113:9596")
	if err != nil {
		panic("dail eth2 failed, err is " + err.Error())
	}

	t.Log("------------- 2")
	//resp, err := eth2Client.BeaconHeaders(context.Background(), constant.HeadBlockIdOfEth2)
	//if err != nil {
	//	t.Fatalf("BeaconHeaders failed err is %v", err)
	//}
	//t.Log("resp", resp)
	//
	//lightResp, err := eth2Client.LightClientUpdate(context.Background(), 618)
	//if err != nil {
	//	t.Errorf("LightClientUpdate failed err is %v", err)
	//}
	//t.Log("lightResp", lightResp)
	//
	//finalityUpdateResp, err := eth2Client.FinallyUpdate(context.Background())
	//if err != nil {
	//	t.Fatalf("FinallyUpdate failed err is %v", err)
	//}
	//t.Log("finalityUpdateResp", finalityUpdateResp)

	blockResp, err := eth2Client.GetBlocks(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetBlocks failed err is %v", err)
	}
	t.Log("blockResp", blockResp)
}

func Test_Key(t *testing.T) {
	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(0))
	fmt.Println("index=0,length=3,hex=", "0x"+common.Bytes2Hex(key2Hex(key, 3)))
	fmt.Println("index=0,length=1,hex=", "0x"+common.Bytes2Hex(key2Hex(key, 1)))
	var key1 []byte
	key1 = rlp.AppendUint64(key1[:0], uint64(190))
	fmt.Println("index=190,length=5,hex=", "0x"+common.Bytes2Hex(key2Hex(key1, 5)))
}

func Test_Redis(t *testing.T) {
	//fmt.Println("0x" + common.Bytes2Hex([]byte("mcs_token_0")))
	//fmt.Println("0x" + common.Bytes2Hex([]byte("zmmap.testnet")))
	redis.Init("redis://:F6U3gV0L6Xwyw1Ko@46.137.199.126:6379/0")
	bytes, err := ioutil.ReadFile("./json.txt")
	if err != nil {
		t.Fatalf("readFile failed err is %v", err)
	}
	redis.GetClient().LPush(context.Background(), redis.ListKey, string(bytes))
}

func Test_NearMcs(t *testing.T) {
	bytes, err := ioutil.ReadFile("./hashjson.txt")
	if err != nil {
		t.Fatalf("readFile failed err is %v", err)
	}
	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		t.Fatalf("unMarshal failed, err %v", err)
	}
	target := make([]mapprotocol.IndexerExecutionOutcomeWithReceipt, 0)
	for _, shard := range data.Shards {
		for _, outcome := range shard.ReceiptExecutionOutcomes {
			if outcome.ExecutionOutcome.Outcome.ExecutorID != "mcs.pandarr.testnet" { // 合约地址
				continue
			}
			if len(outcome.ExecutionOutcome.Outcome.Logs) == 0 {
				continue
			}
			for _, ls := range outcome.ExecutionOutcome.Outcome.Logs {
				//splits := strings.Split(ls, ":")
				//if len(splits) != 2 {
				//	continue
				//}
				//if !ExistInSlice(splits[0], mapprotocol.NearEventType) {
				//	continue
				//}
				//t.Logf("log is %v", ls)
				if !strings.HasPrefix(ls, mapprotocol.HashOfTransferOut) && !strings.HasPrefix(ls, mapprotocol.HashOfDepositOut) {
					continue
				}
			}

			target = append(target, outcome)
		}
	}

	if len(target) == 0 {
		t.Logf("data is %+v", data)
		return
	}

	cli, err := nearclient.NewClient("https://archival-rpc.testnet.near.org")
	if err != nil {
		t.Fatalf("unMarshal failed, err %v", err)
	}
	for _, tg := range target {
		// get fromChainId and toChainId
		logs := strings.SplitN(tg.ExecutionOutcome.Outcome.Logs[0], ":", 2)
		t.Logf("tg %+v ", logs)
		t.Logf("tg %+v ", logs[1])
		out := near.TransferOut{}
		err = json.Unmarshal([]byte(logs[1]), &out)
		if err != nil {
			t.Fatalf("logs format failed %v", err)
		}
		fmt.Println("out.to", out.ToChain)
		fmt.Println("out.from", out.FromChain)
		blk, err := cli.NextLightClientBlock(context.Background(), tg.ExecutionOutcome.BlockHash)
		if err != nil {
			t.Errorf("NextLightClientBlock failed, err %v", err)
		}

		clientHead, err := cli.BlockDetails(context.Background(), block.BlockID(blk.InnerLite.Height))
		if err != nil {
			t.Errorf("BlockDetails failed, err %v", err)
		}

		fmt.Printf("clientHead hash is %v \n", clientHead.Header.Hash)

		proof, err := cli.LightClientProof(context.Background(), nearclient.Receipt{
			ReceiptID:       tg.ExecutionOutcome.ID,
			ReceiverID:      tg.Receipt.ReceiverID,
			LightClientHead: clientHead.Header.Hash,
		})
		if err != nil {
			t.Errorf("LightClientProof failed, err %v", err)
		}

		d, _ := json.Marshal(blk)
		p, _ := json.Marshal(proof)
		t.Logf("block %+v, \n proof %+v \n", string(d), string(p))

		blkBytes := near.Borshify(blk)
		t.Logf("blockBytes, 0x%v", common.Bytes2Hex(blkBytes))
		proofBytes, err := near.BorshifyOutcomeProof(proof)
		if err != nil {
			t.Fatalf("borshifyOutcomeProof failed, err is %v", proofBytes)
		}
		t.Logf("proofBytes, 0x%v", common.Bytes2Hex(proofBytes))

		all, err := mapprotocol.Near.Methods["getBytes"].Inputs.Pack(blkBytes, proofBytes)
		if err != nil {
			t.Fatalf("getBytes failed, err is %v", err)
		}

		input, err := mapprotocol.Near.Pack(mapprotocol.MethodVerifyProofData, all)
		if err != nil {
			t.Fatalf("getBytes failed, err is %v", err)
		}

		fmt.Println("请求参数 ---------- ", "0x"+common.Bytes2Hex(all))
		fmt.Println("请求参数 ---------- input ", "0x"+common.Bytes2Hex(input))
		err = call(input, mapprotocol.Near, mapprotocol.MethodVerifyProofData)
		if err != nil {
			t.Fatalf("call failed, err is %v", err)
		}
	}
	//t.Logf("data is %+v", data)
}

func call(input []byte, useAbi abi.ABI, method string) error {
	to := common.HexToAddress("0xa44b62879B9FfE422615CBccB993E090D93fD0eE")
	count := 0
	for {
		count++
		if count == 5 {
			return errors.New("retry is full")
		}
		outPut, err := dialMapConn().CallContract(context.Background(),
			ethereum.CallMsg{
				From: common.HexToAddress("0xE0DC8D7f134d0A79019BEF9C2fd4b2013a64fCD6"),
				To:   &to,
				Data: input,
			},
			nil,
		)
		if err != nil {
			log.Printf("callContract failed, err is %v \n", err)
			if strings.Index(err.Error(), "read: connection reset by peer") != -1 {
				log.Printf("err is : %s, will retry \n", err)
				time.Sleep(time.Second * 5)
				continue
			}
			return err
		}

		resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
		if err != nil {
			return err
		}

		ret := struct {
			Success bool
			Message string
			Logs    []byte
		}{}

		err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
		if err != nil {
			return err
		}
		if !ret.Success {
			return fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
		}
		if ret.Success == true {
			log.Println("mcs verify log success", "success", ret.Success)
			log.Println("mcs verify log success", "logs", "0x"+common.Bytes2Hex(ret.Logs))
			return nil
		}
	}
}

func ExistInSlice(target string, dst []string) bool {
	for _, d := range dst {
		if target == d {
			return true
		}
	}

	return false
}

var ContractAddr = common.HexToAddress("0xA7D3A66013DE32f0a44C92E337Af22C4344a2d62")

func dialConn() *ethclient.Client {
	conn, err := ethclient.Dial("http://18.142.54.137:7445")
	if err != nil {
		log.Fatalf("Failed to connect to the atlas: %v", err)
	}
	return conn
}

func dialMapConn() *ethclient.Client {
	conn, err := ethclient.Dial("http://18.142.54.137:7445")
	if err != nil {
		log.Fatalf("Failed to connect to the atlas: %v", err)
	}
	return conn
}

func TestLoadPrivate(t *testing.T) {
	path := "/Users/t/data/atlas-1/keystore/UTC--2022-06-07T04-22-55.836701000Z--f9803e9021e56e68662351fe43773934c4a276b8"
	password := ""
	addr, private := LoadPrivate(path, password)
	fmt.Println("============================== addr: ", addr)
	fmt.Printf("============================== private key: %x\n", crypto.FromECDSA(private))
}

func getMapTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.MAPBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	fmt.Println("block.Transactions() ------------- ", len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs, nil
}

func getReceiptsByTxsHash(conn *ethclient.Client, txsHash []common.Hash) ([]*types.Receipt, error) {
	rs := make([]*types.Receipt, 0, len(txsHash))
	for _, h := range txsHash {
		r, err := conn.TransactionReceipt(context.Background(), h)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func getLastReceipt(conn *ethclient.Client, latestBlock *big.Int) (*types.Receipt, error) {
	fmt.Println("--------------------------------- 进来了 ")
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
			TxHash:      common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			TxIndex:     ll.TxIndex,
			BlockHash:   common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			Index:       ll.Index,
			Removed:     ll.Removed,
		})
		tl := ll
		el = append(el, &tl)
	}
	d, _ := json.Marshal(rl)
	fmt.Println("ddd ------------------ ", string(d))
	receipt.Logs = rl
	receipt.Bloom = maptypes.CreateBloom(maptypes.Receipts{receipt})
	fmt.Println("receipt.Bloom ------------------ ", receipt.Bloom)
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

// deriveBufferPool holds temporary encoder buffers for DeriveSha and TX encoding.
var encodeBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

type DerivableList interface {
	Len() int
	EncodeIndex(int, *bytes.Buffer)
}

func encodeForDerive(list DerivableList, i int, buf *bytes.Buffer) []byte {
	buf.Reset()
	list.EncodeIndex(i, buf)
	// It's really unfortunate that we need to do perform this copy.
	// StackTrie holds onto the values until Hash is called, so the values
	// written to it must not alias.
	return common.CopyBytes(buf.Bytes())
}

func DeriveTire(rs types.Receipts, tr *trie.Trie) *trie.Trie {
	valueBuf := encodeBufferPool.Get().(*bytes.Buffer)
	defer encodeBufferPool.Put(valueBuf)

	var indexBuf []byte
	for i := 1; i < rs.Len() && i <= 0x7f; i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	if rs.Len() > 0 {
		indexBuf = rlp.AppendUint64(indexBuf[:0], 0)
		value := encodeForDerive(rs, 0, valueBuf)
		tr.Update(indexBuf, value)
	}
	for i := 0x80; i < rs.Len(); i++ {
		indexBuf = rlp.AppendUint64(indexBuf[:0], uint64(i))
		value := encodeForDerive(rs, i, valueBuf)
		tr.Update(indexBuf, value)
	}
	return tr
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	fmt.Println("hash ------------------- ", tr.Hash())
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}

	return proof, nil
}

func TestUpdateHeader(t *testing.T) {
	cli := dialConn()
	header, err := cli.MAPHeaderByNumber(context.Background(), new(big.Int).SetUint64(1156000))
	if err != nil {
		t.Fatal("err ", err)
	}
	t.Log("online header.ReceiptHash", header.ReceiptHash)

	txsHash, err := getMapTransactionsHashByBlockNumber(cli, new(big.Int).SetUint64(1156000))
	if err != nil {
		t.Fatal("err ", fmt.Errorf("idSame unable to get tx hashes Logs: %w", err))
	}

	receipts, err := getReceiptsByTxsHash(cli, txsHash)
	if err != nil {
		t.Fatal("err ", fmt.Errorf("unable to get receipts hashes Logs: %w", err))
	}
	fmt.Println("receipts ", len(receipts))

	lr, err := getLastReceipt(cli, new(big.Int).SetUint64(1156000))
	if err != nil {
		t.Fatal("err ", fmt.Errorf("unable to get last receipts in epoch last %w", err))
	}
	receipts = append(receipts, lr)
	t.Log("receipts ----------- ", len(receipts))
	t.Log("third ------------- ", types.DeriveSha(types.Receipts(receipts), trie.NewStackTrie(nil)))
	//for i := 1; i < 21; i++ {
	//	number := int64(i * 1000)
	//	fmt.Println("============================== number: ", number)
	//	header, err := cli.MAPHeaderByNumber(context.Background(), big.NewInt(number))
	//	if err != nil {
	//		t.Fatalf(err.Error())
	//	}
	//
	//	h := mapprotocol.ConvertHeader(header)
	//	aggPK, _, _, err := mapprotocol.GetAggPK(cli, header.Number, header.Extra)
	//	if err != nil {
	//		t.Fatalf(err.Error())
	//	}
	//
	//	//printHeader(header)
	//	//printAggPK(aggPK)
	//	//_ = h
	//
	//	input, err := mapprotocol.PackInput(mapprotocol.LightManger, mapprotocol.MethodUpdateBlockHeader, h, aggPK)
	//	if err != nil {
	//		t.Fatalf(err.Error())
	//	}
	//
	//	path := "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-15T07-51-25.301943000Z--e0dc8d7f134d0a79019bef9c2fd4b2013a64fcd6"
	//	password := "1234"
	//	from, private := LoadPrivate(path, password)
	//	if err := SendContractTransaction(cli, from, ContractAddr, nil, private, input); err != nil {
	//		t.Fatalf(err.Error())
	//	}
	//}
}

func TestVerifyProofData(t *testing.T) {
	var (
		number = big.NewInt(106020)
		//number       = big.NewInt(4108)
		//number       = big.NewInt(55342)
		txIndex uint = 0
	)
	cli := dialMapConn()

	header, err := cli.MAPHeaderByNumber(context.Background(), number)
	if err != nil {
		t.Fatalf(err.Error())
	}

	txsHash, err := getTransactionsHashByBlockNumber(cli, number)
	if err != nil {
		t.Fatalf(err.Error())
	}
	receipts, err := getReceiptsByTxsHash(cli, txsHash)
	if err != nil {
		t.Fatalf(err.Error())
	}
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])

	proof, err := getProof(receipts, txIndex)
	if err != nil {
		t.Fatalf(err.Error())
	}

	aggPK, _, _, err := mapprotocol.GetAggPK(cli, header.Number, header.Extra)
	if err != nil {
		t.Fatalf(err.Error())
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))

	//fmt.Println("============================== number: ", number)
	//printHeader(header)
	//printAggPK(aggPK)
	//printReceipt(receipt)
	//fmt.Println("============================== KeyIndex: ", "0x"+common.Bytes2Hex(key))
	//printProof(proof)

	rp := mapprotocol.ReceiptProof{
		Header:   mapprotocol.ConvertHeader(header),
		AggPk:    aggPK,
		Receipt:  receipt,
		KeyIndex: key,
		Proof:    proof,
	}

	input, err := mapprotocol.PackInput(mapprotocol.Near, mapprotocol.MethodVerifyProofData, rp)
	if err != nil {
		t.Fatalf(err.Error())
	}
	path := "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-15T07-51-25.301943000Z--e0dc8d7f134d0a79019bef9c2fd4b2013a64fcd6"
	password := "1234"
	from, _ := LoadPrivate(path, password)
	output, err := dialConn().CallContract(context.Background(), eth.CallMsg{From: from, To: &ContractAddr, Data: input}, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	resp, err := mapprotocol.Near.Methods[mapprotocol.MethodVerifyProofData].Outputs.Unpack(output)
	if err != nil {
		t.Fatalf(err.Error())
	}

	ret := struct {
		Success bool
		Message string
	}{}
	if err := mapprotocol.LightManger.Methods[mapprotocol.MethodVerifyProofData].Outputs.Copy(&ret, resp); err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Printf("============================== success: %v, message: %v\n", ret.Success, ret.Message)
}

func TestGetLog(t *testing.T) {
	//number       = big.NewInt(4108)
	//number       = big.NewInt(55342)
	var MapTransferOut utils.EventSig = "mapTransferOut(bytes,bytes,bytes32,uint256,uint256,bytes,uint256,bytes)"
	query := buildQuery(common.HexToAddress("0xf03aDB732FBa8Fca38C00253B1A1aa72CCA026E6"),
		MapTransferOut, big.NewInt(106020), big.NewInt(106020))

	// querying for logs
	logs, err := dialConn().FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("unable to Filter Logs: %s", err)
	}
	t.Logf("log len is %v", len(logs))
}

// buildQuery constructs a query for the bridgeContract by hashing sig to get the event topic
func buildQuery(contract common.Address, sig utils.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{contract},
		Topics: [][]common.Hash{
			{sig.GetTopic()},
		},
	}
	return query
}

func TestHashStatus(t *testing.T) {
	hash := common.HexToHash("0xd4eadb06f544ffc8263d550f38fd4a52e882919b8cef1716ebbbed4294773a93")
	hash1 := common.HexToHash("0x2d9fd9553b1419cb2ee690788def96eedb6ecada203798bca984430461fd7144")
	client := dialConn()
	for {
		time.Sleep(time.Millisecond * 1000)
		receipt, err := client.TransactionReceipt(context.Background(), hash)
		if err != nil {
			break
		}
		receipt1, err := client.TransactionReceipt(context.Background(), hash1)
		if err != nil {
			break
		}

		t.Log("Tx is successful", "tx", hash, "receipt.Status", receipt.Status)
		t.Log("Tx is successful", "tx", hash1, "receipt.Status", receipt1.Status)
		break
	}
}

func SendContractTransaction(client *ethclient.Client, from, toAddress common.Address, value *big.Int, privateKey *ecdsa.PrivateKey, input []byte) error {
	// Ensure a valid value field and resolve the account nonce
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		return err
	}
	//fmt.Printf("============================== from: %s, nonce: %d\n", from, nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	gasLimit := uint64(2100000) // in units
	// If the contract surely has code (or code is not needed), estimate the transaction
	msg := eth.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	gasLimit, err = client.EstimateGas(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("contract exec failed, %s", err.Error())
	}
	if gasLimit < 1 {
		gasLimit = 866328
	}

	// Create the transaction, sign it and schedule it for execution
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, input)

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return err
	}
	//fmt.Println("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	txHash := signedTx.Hash()
	count := 0
	for {
		time.Sleep(time.Millisecond * 1000)
		_, isPending, err := client.TransactionByHash(context.Background(), txHash)
		if err != nil {
			return err
		}
		count++
		if !isPending {
			break
		} else {
			log.Println("======================== pending...")
		}
	}
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		logs, _ := json.Marshal(receipt.Logs)
		log.Printf("Transaction Success, number: %v, hash: %v， logs: %v\n", receipt.BlockNumber.Uint64(), receipt.BlockHash, string(logs))
	} else if receipt.Status == types.ReceiptStatusFailed {
		log.Println("Transaction Failed. ", "block number: ", receipt.BlockNumber.Uint64())
		return errors.New("transaction failed")
	}
	return nil
}

func LoadPrivate(path, password string) (common.Address, *ecdsa.PrivateKey) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	key, err := keystore.DecryptKey(bs, password)
	if err != nil || key == nil {
		panic(fmt.Errorf("error decrypting key: %v", err))
	}
	priKey := key.PrivateKey
	addr := crypto.PubkeyToAddress(priKey.PublicKey)

	if priKey == nil {
		panic("load privateKey failed")
	}
	return addr, priKey
}

func printHeader(header *maptypes.Header) {
	type blockHeader struct {
		ParentHash  string
		Coinbase    string
		Root        string
		TxHash      string
		ReceiptHash string
		Bloom       string
		Number      *big.Int
		GasLimit    *big.Int
		GasUsed     *big.Int
		Time        *big.Int
		ExtraData   string
		MixDigest   string
		Nonce       string
		BaseFee     *big.Int
	}
	h := blockHeader{
		ParentHash:  "0x" + common.Bytes2Hex(header.ParentHash[:]),
		Coinbase:    header.Coinbase.String(),
		Root:        "0x" + common.Bytes2Hex(header.Root[:]),
		TxHash:      "0x" + common.Bytes2Hex(header.TxHash[:]),
		ReceiptHash: "0x" + common.Bytes2Hex(header.ReceiptHash[:]),
		Bloom:       "0x" + common.Bytes2Hex(header.Bloom[:]),
		Number:      header.Number,
		GasLimit:    new(big.Int).SetUint64(header.GasLimit),
		GasUsed:     new(big.Int).SetUint64(header.GasUsed),
		Time:        new(big.Int).SetUint64(header.Time),
		ExtraData:   "0x" + common.Bytes2Hex(header.Extra),
		MixDigest:   "0x" + common.Bytes2Hex(header.MixDigest[:]),
		Nonce:       "0x" + common.Bytes2Hex(header.Nonce[:]),
		BaseFee:     header.BaseFee,
	}
	fmt.Printf("============================== header: %+v\n", h)
}

func printAggPK(aggPk *mapprotocol.G2) {
	type G2Str struct {
		xr string
		xi string
		yr string
		yi string
	}
	g2 := G2Str{
		xr: "0x" + common.Bytes2Hex(aggPk.Xr.Bytes()),
		xi: "0x" + common.Bytes2Hex(aggPk.Xi.Bytes()),
		yr: "0x" + common.Bytes2Hex(aggPk.Yr.Bytes()),
		yi: "0x" + common.Bytes2Hex(aggPk.Yi.Bytes()),
	}
	fmt.Printf("============================== aggPk: %+v\n", g2)
}

func printReceipt(r *mapprotocol.TxReceipt) {
	type txLog struct {
		Addr   common.Address
		Topics []string
		Data   string
	}

	type receipt struct {
		ReceiptType       *big.Int
		PostStateOrStatus string
		CumulativeGasUsed *big.Int
		Bloom             string
		Logs              []txLog
	}

	logs := make([]txLog, 0, len(r.Logs))
	for _, lg := range r.Logs {
		topics := make([]string, 0, len(lg.Topics))
		for _, tp := range lg.Topics {
			topics = append(topics, "0x"+common.Bytes2Hex(tp))
		}
		logs = append(logs, txLog{
			Addr:   lg.Addr,
			Topics: topics,
			Data:   "0x" + common.Bytes2Hex(lg.Data),
		})
	}

	rr := receipt{
		ReceiptType:       r.ReceiptType,
		PostStateOrStatus: "0x" + common.Bytes2Hex(r.PostStateOrStatus),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             "0x" + common.Bytes2Hex(r.Bloom),
		Logs:              logs,
	}
	fmt.Printf("============================== Receipt: %+v\n", rr)
}

func printProof(proof [][]byte) {
	p := make([]string, 0, len(proof))
	for _, v := range proof {
		p = append(p, "0x"+common.Bytes2Hex(v))
	}
	fmt.Println("============================== proof: ", p)
}

func getTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.MAPBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs, nil
}

func keyBytesToHex(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	//nibbles[l-1] = 16
	return nibbles
}

func key2Hex(str []byte, proofLength int) []byte {
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
