package writer

import (
	"context"
	"math/big"

	"github.com/mapprotocol/compass/internal/constant"

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
	w.log.Info("Attempting to resolve message", "type", m.Type, "src", m.Source, "dst", m.Destination)

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
	gasLimit, err := w.conn.Client().EstimateGas(context.Background(), msg)
	if err != nil {
		w.log.Error("EstimateGas failed sendTx", "error:", err.Error())
		return nil, err
	}

	w.log.Info("sendTx gasPrice ", "gasPrice", gasPrice,
		"w.conn.Opts().GasTipCap", w.conn.Opts().GasTipCap, "w.conn.Opts().GasFeeCap", w.conn.Opts().GasFeeCap)
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

func (w *Writer) needNonce(err error) bool {
	if err == nil || err.Error() == constant.ErrNonceTooLow.Error() {
		return true
	}

	return false
}
