package writer

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/msg"
)

type Writer struct {
	cfg     chain.Config
	conn    chain.Connection
	log     log15.Logger
	stop    <-chan int
	sysErr  chan<- error // Reports fatal error to core
	metrics *metrics.ChainMetrics
}

// New creates and returns Writer
func New(conn chain.Connection, cfg *chain.Config, log log15.Logger, stop <-chan int, sysErr chan<- error,
	m *metrics.ChainMetrics) *Writer {
	return &Writer{
		cfg:     *cfg,
		conn:    conn,
		log:     log,
		stop:    stop,
		sysErr:  sysErr,
		metrics: m,
	}
}

func (w *Writer) start() error {
	w.log.Debug("Starting Writer...")
	return nil
}

// ResolveMessage handles any given message based on type
// A bool is returned to indicate failure/success, this should be ignored except for within tests.
func (w *Writer) ResolveMessage(m msg.Message) bool {
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination, "nonce", m.DepositNonce)

	switch m.Type {
	case msg.SyncToMap:
		return w.execToMapMsg(m)
	case msg.SyncFromMap:
		return w.execMap2OtherMsg(m)
	case msg.SwapTransfer:
		fallthrough
	case msg.SwapWithProof:
		fallthrough
	case msg.SwapWithMapProof:
		// same process
		return w.exeSwapMsg(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

// sendTx send tx to an address with value and input data
func (w *Writer) sendTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
	gasPrice := w.conn.Opts().GasPrice
	nonce := w.conn.Opts().Nonce
	from := w.conn.Keypair().CommonAddress()

	msg := ethereum.CallMsg{
		From:     from,
		To:       toAddress,
		GasPrice: gasPrice,
		Value:    value,
		Data:     input,
	}
	w.log.Debug("eth CallMsg", "msg", msg)
	w.log.Debug("eth CallMsg", "toAddress", toAddress)
	gasLimit, err := w.conn.Client().EstimateGas(context.Background(), msg)
	if err != nil {
		w.log.Error("EstimateGas failed sendTx", "error:", err.Error())
		return nil, err
	}

	w.log.Info("gasPrice ---------------------- ", "gasPrice", gasPrice)
	// td interface
	var td types.TxData
	// EIP-1559
	if gasPrice != nil {
		// legacy branch
		td = &types.LegacyTx{
			Nonce:    nonce.Uint64(),
			Value:    value,
			To:       toAddress,
			Gas:      gasLimit,
			GasPrice: gasPrice,
			Data:     input,
		}
	} else {
		// london branch
		td = &types.DynamicFeeTx{
			Nonce:     nonce.Uint64(),
			Value:     value,
			To:        toAddress,
			Gas:       gasLimit,
			GasTipCap: w.conn.Opts().GasTipCap,
			GasFeeCap: w.conn.Opts().GasFeeCap,
			Data:      input,
		}
	}

	tx := types.NewTx(td)
	chainID := big.NewInt(int64(w.cfg.Id))
	privateKey := w.conn.Keypair().PrivateKey()

	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), privateKey)
	if err != nil {
		w.log.Error("SignTx failed", "error:", err.Error())
		return nil, err
	}

	err = w.conn.Client().SendTransaction(context.Background(), signedTx)
	if err != nil {
		w.log.Error("SendTransaction failed", "error:", err.Error())
		return nil, err
	}
	return signedTx, nil
}

// this function will block for the txhash given
func (w *Writer) blockForPending(txHash common.Hash) error {
	for {
		_, isPending, err := w.conn.Client().TransactionByHash(context.Background(), txHash)
		if err != nil {
			w.log.Info("blockForPending tx is temporary not found", "err is not found?", errors.Is(err, errors.New("not found")), "err", err)
			if strings.Index(err.Error(), "not found") != -1 {
				w.log.Info("tx is temporary not found, please wait...", "tx", txHash)
				time.Sleep(time.Millisecond * 900)
				continue
			}
			return err
		}

		if isPending {
			w.log.Info("tx is pending, please wait...")
			time.Sleep(time.Millisecond * 900)
			continue
		}
		w.log.Info("tx is successful", "tx", txHash)
		break
	}
	return nil
}
