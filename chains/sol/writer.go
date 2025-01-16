package sol

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/mapprotocol/compass/internal/butter"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/pkg/errors"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/msg"
	"github.com/mapprotocol/compass/pkg/util"
)

type Writer struct {
	cfg    *Config
	log    log15.Logger
	conn   *Connection
	stop   <-chan int
	sysErr chan<- error
}

func newWriter(conn *Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error) *Writer {
	return &Writer{
		cfg:    cfg,
		conn:   conn,
		log:    log,
		stop:   stop,
		sysErr: sysErr,
	}
}

func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination)
	switch m.Type {
	case msg.SwapSolProof:
		return w.exeMcs(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

type GenerateRequest struct {
	LogAddr      string   `json:"logAddr"`
	LogTopics    []string `json:"logTopics"`
	LogData      string   `json:"logData"`
	Signatures   []string `json:"signatures"`
	OraclePacked string   `json:"oraclePacked"`
	Relayer      string   `json:"relayer"`
}

func (w *Writer) exeMcs(m msg.Message) bool {
	var (
		errorCount  int64
		firstFinish bool
		log         = m.Payload[0].(*types.Log)
		sign        = m.Payload[1].([][]byte)
		method      = m.Payload[2].(string)
	)

	fmt.Println("log ------------ ", common.Bytes2Hex(log.Data))
	data, err := w.generateData(log, sign)
	if err != nil {
		w.log.Error("Error generating data", "error", err)
		return false
	}

	time.Sleep(time.Minute)
	for {
		select {
		case <-w.stop:
			return false
		default:
			w.log.Info("Send transaction", "srcHash", log.TxHash, "method", method)
			mcsTx, err := w.sendTx(data.Data.Tx[0])
			if err == nil {
				w.log.Info("Submitted cross tx execution", "src", m.Source, "dst", m.Destination, "srcHash", log.TxHash, "mcsTx", mcsTx)
				err = w.txStatus(*mcsTx)
				if err == nil {
					w.log.Info("TxHash status is successful, will next tx")
					firstFinish = true
				}
				w.log.Error("TxHash status is successful, will next tx")
			} else if w.cfg.SkipError && errorCount >= 9 {
				w.log.Warn("Execution failed, ignore this error, Continue to the next ", "srcHash", log.TxHash, "err", err)
				m.DoneCh <- struct{}{}
				return true
			} else {
				for e := range constant.IgnoreError {
					if strings.Index(err.Error(), e) != -1 {
						w.log.Info("Ignore This Error, Continue to the next", "id", m.Destination, "err", err)
						m.DoneCh <- struct{}{}
						return true
					}
				}
				w.log.Warn("Execution failed, will retry", "srcHash", log.TxHash, "err", err)
			}
			errorCount++
			if errorCount >= 10 && !firstFinish {
				w.mosAlarm(log.TxHash, err)
				errorCount = 0
			}
			time.Sleep(constant.TxRetryInterval)
			if data.Data.SwapData.ToToken == "" && data.Data.SwapData.ToAddress == "" && data.Data.SwapData.MinAmountOutBN == "" && firstFinish {
				m.DoneCh <- struct{}{}
				return true
			}
			// have swap
			for {
				select {
				case <-w.stop:
					return false
				default:
					// 请求solana
					//w.solCrossIn(data.Data.SwapData.ToToken, data.Data.SwapData.ToAddress)
				}
			}

		}
	}
}

func (w *Writer) generateData(log *types.Log, sign [][]byte) (*Resp, error) {
	// 构建http request
	mulInfo, err := chain.MulSignInfo(0, uint64(w.cfg.MapChainID))
	if err != nil {
		return nil, errors.Wrap(err, "failed to mul sign info")
	}
	receipt, err := chain.GenLogReceipt(log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate receipt")
	}
	pack, err := mapprotocol.PackAbi.Methods[mapprotocol.MethodOfSolidityPack].Inputs.Pack(receipt,
		mulInfo.Version, big.NewInt(0).SetUint64(log.BlockNumber), big.NewInt(int64(w.cfg.MapChainID)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack abi")
	}

	topic := make([]string, 0)
	for _, ele := range log.Topics {
		topic = append(topic, ele.String())
	}
	signs := make([]string, 0)
	for _, ele := range sign {
		signs = append(signs, "0x"+common.Bytes2Hex(ele))
	}

	base58, err := solana.PrivateKeyFromBase58(w.cfg.Pri)
	if err != nil {
		return nil, errors.Wrap(err, "failed to privateKeyFromBase58")
	}

	request := GenerateRequest{
		LogAddr:      log.Address.String(),
		LogTopics:    topic,
		LogData:      "0x" + common.Bytes2Hex(log.Data),
		Signatures:   signs,
		OraclePacked: "0x" + common.Bytes2Hex(pack),
		Relayer:      base58.PublicKey().String(),
	}
	reqData, _ := json.Marshal(&request)
	resp, err := http.Post(w.cfg.MessageIn, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		return nil, errors.Wrap(err, "Error Post data")
	}
	defer resp.Body.Close()
	readAll, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	w.log.Info("Receipt messageIn body is", "body", string(readAll))
	ret := Resp{}
	err = json.Unmarshal(readAll, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}
	if ret.Code != 0 {
		return nil, fmt.Errorf("resp code is %v", ret.Code)
	}
	return &ret, nil
}

func (w *Writer) sendTx(data string) (*solana.Signature, error) {
	w.log.Info("Sending transaction", "data", data)
	bbs, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}
	trx, err := solana.TransactionFromBytes(bbs)
	if err != nil {
		return nil, err
	}
	resp, err := w.conn.cli.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	trx.Message.RecentBlockhash = resp.Value.Blockhash
	w.log.Info("Sending transaction", "blockHash", resp.Value.Blockhash)
	signPri, _ := solana.PrivateKeyFromBase58(w.cfg.Pri)
	// sign
	_, err = trx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key == signPri.PublicKey() {
			return &signPri
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	w.log.Info("Sending will transaction")

	//maxRetries := uint(5)
	//minContextSlot := uint64(1)
	sig, err := w.conn.cli.SendTransactionWithOpts(context.TODO(), trx, rpc.TransactionOpts{
		SkipPreflight: false,
		//MaxRetries:     &maxRetries,
		//MinContextSlot: &minContextSlot,
		// PreflightCommitment: rpc.CommitmentProcessed, 第二笔交易
	})
	if err != nil {
		return nil, err

	}
	return &sig, nil
}

func (w *Writer) txStatus(txHash solana.Signature) error {
	var count int64
	time.Sleep(time.Second * 2)
	for {
		version := uint64(0)
		_, err := w.conn.cli.GetTransaction(context.Background(), txHash, &rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase58,
			Commitment:                     rpc.CommitmentFinalized,
			MaxSupportedTransactionVersion: &version,
		})
		if err != nil {
			count++
			w.log.Error("Failed to GetTransaction", "err", err)
			time.Sleep(5 * time.Second)
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			continue
		}

		txResult, err := w.conn.cli.GetSignatureStatuses(context.Background(), true, txHash)
		if err != nil {
			count++
			w.log.Error("Failed to GetSignatureStatuses", "err", err)
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			time.Sleep(5 * time.Second)
			continue
		}
		fmt.Println("txResult ------------------ ", txResult)
		if txResult.Value == nil || len(txResult.Value) == 0 || txResult.Value[0].Err != nil {
			count++
			if count >= 30 {
				return errors.New("The Tx pending state is too long")
			}
			continue
		}

		w.log.Info("Tx receipt status is success", "hash", txHash)
		return nil
	}
}

func (w *Writer) mosAlarm(tx interface{}, err error) {
	util.Alarm(context.Background(), fmt.Sprintf("mos map2tron failed, srcHash=%v err is %s", tx, err.Error()))
}

func (w *Writer) solCrossIn(toToken, receiver, minAmount string, l *types.Log) (*butter.SolCrossInResp, error) {
	signPri, _ := solana.PrivateKeyFromBase58(w.cfg.Pri)
	router := signPri.PublicKey().String()
	orderId := l.Topics[1]

	query := fmt.Sprintf("fromChainId=%s&chainPoolChain=%s&"+
		"chainPoolTokenAddress=%s&chainPoolTokenAmount=%s&"+
		"tokenOutAddress=%s&slippage=%d&"+
		"router=%s&minAmountOut=%s&from=%s&orderIdHex=%s&receiver=%s&feeRatio=%s",
		"22776", "22776",
		"param.RelayToken", "param.RelayAmount",
		toToken, 100,
		router, minAmount, "param.Sender", orderId.Hex(), receiver, "200",
	)
	data, err := butter.SolCrossIn(w.cfg.ButterHost, query)
	if err != nil {
		w.log.Error("Failed to butter.SolCrossIn", "err", err)
		return nil, err
	}

	return data, nil
}

type Resp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Tx       []string `json:"tx"`
		SwapData struct {
			ToToken        string `json:"to_token"`
			ToAddress      string `json:"to_address"`
			MinAmountOutBN string `json:"minAmountOutBN"`
		} `json:"swap_data"`
	} `json:"data"`
}

func DecodeRelayData(data string) (*MessageData, error) {
	// Decode hex string
	bytesData, err := hex.DecodeString(data)
	if err != nil {
		return nil, err
	}

	nAbi, err := abi.JSON(strings.NewReader(`[{"inputs":[{"internalType":"bytes","name":"","type":"bytes"}],"name":"soliditypack","outputs":[],"stateMutability":"nonpayable","type":"function"}]`))
	if err != nil {
		return nil, err
	}
	unpack, err := nAbi.Methods["soliditypack"].Inputs.Unpack(bytesData)
	if err != nil {
		return nil, err
	}

	bytesData = unpack[0].([]byte)
	// Helper function to parse a big.Int from a substring
	parseBigInt := func(start, length int) *big.Int {
		substr := bytesData[start : start+length]
		return new(big.Int).SetBytes(substr)
	}

	ret := &MessageData{}
	// Extract values based on offsets
	version := parseBigInt(0, 1)
	ret.Version = version

	messageType := parseBigInt(1, 1)
	ret.MessageType = messageType

	tokenLen := parseBigInt(2, 1)
	ret.TokenLen = tokenLen

	mosLen := parseBigInt(3, 1)
	ret.MosLen = mosLen

	fromLen := parseBigInt(4, 1)
	ret.FromLen = fromLen

	toLen := parseBigInt(5, 1)
	ret.ToLen = toLen

	payloadLen := parseBigInt(6, 2)
	ret.PayloadLen = payloadLen

	tokenAmount := parseBigInt(16, 16)
	ret.TokenAmount = tokenAmount

	// Calculate dynamic offsets
	start := 32
	end := start + int(tokenLen.Int64())
	//tokenAddress := hex.EncodeToString(bytesData[start:end])
	ret.TokenAddress = bytesData[start:end]

	start = end
	end = start + int(mosLen.Int64())
	ret.Mos = bytesData[start:end]

	start = end
	end = start + int(fromLen.Int64())
	ret.From = bytesData[start:end]

	start = end
	end = start + int(toLen.Int64())
	ret.To = bytesData[start:end]

	start = end
	ret.Payload = bytesData[start:end]
	return ret, nil
}

type MessageData struct {
	Version      *big.Int
	MessageType  *big.Int
	TokenLen     *big.Int
	MosLen       *big.Int
	FromLen      *big.Int
	ToLen        *big.Int
	PayloadLen   *big.Int
	TokenAmount  *big.Int
	TokenAddress []byte
	Mos          []byte
	From         []byte
	To           []byte
	Payload      []byte
}
