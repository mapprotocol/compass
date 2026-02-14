package chain

import (
	"context"
	"math/big"
	"strings"

	"github.com/mapprotocol/compass/pkg/msg"

	"github.com/mapprotocol/compass/core"

	"github.com/mapprotocol/compass/internal/constant"

	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Writer struct {
	cfg    Config
	conn   core.Connection
	log    log15.Logger
	stop   <-chan int
	sysErr chan<- error // Reports fatal error to core
}

// NewWriter creates and returns Writer
func NewWriter(conn core.Connection, cfg *Config, log log15.Logger, stop <-chan int, sysErr chan<- error) *Writer {
	return &Writer{
		cfg:    *cfg,
		conn:   conn,
		log:    log,
		stop:   stop,
		sysErr: sysErr,
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
	case msg.SwapWithProof:
		fallthrough
	case msg.SwapWithMapProof:
		return w.exeSwapMsg(m)
	case msg.SwapWithMerlin:
		return w.merlinWithMsg(m)
	case msg.Proposal:
		return w.proposal(m)
	default:
		w.log.Error("Unknown message type received", "type", m.Type)
		return false
	}
}

// sendTx send tx to an address with value and input data
func (w *Writer) sendTx(toAddress *common.Address, value *big.Int, input []byte) (*types.Transaction, error) {
	gasPrice := w.conn.Opts().GasPrice
	nonce := w.conn.Opts().Nonce
	from := w.conn.Keypair().Address

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
		if err.Error() == "execution reverted" {
			_, serr := w.conn.Client().SelfEstimateGas(context.Background(),
				w.cfg.Endpoint, from.Hex(), toAddress.Hex(), "0x"+common.Bytes2Hex(input))
			w.log.Error("EstimateGas failed sendTx", "error:", serr)
			if serr != nil {
				return nil, serr
			}
		}
		return nil, err
	}

	gasTipCap := w.conn.Opts().GasTipCap
	gasFeeCap := w.conn.Opts().GasFeeCap
	w.log.Info("SendTx gasPrice before", "gasPrice", gasPrice, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap)
	if w.cfg.LimitMultiplier > 1 {
		gasLimit = uint64(float64(gasLimit) * w.cfg.LimitMultiplier)
	}
	if w.cfg.Id == constant.XlayerId {
		gasLimit = gasLimit + 500000
	}
	if w.cfg.Id == constant.LineaChainId {
		gasLimit = gasLimit + 500000
		if gasLimit > 1500000 {
			gasLimit = 1500000
		}
	}
	if w.cfg.GasMultiplier > 1 && gasPrice != nil {
		gasPrice = big.NewInt(0).SetInt64(int64(float64(gasPrice.Int64()) * w.cfg.GasMultiplier))
	}
	if gasLimit < constant.MinGasLimit {
		gasLimit = constant.MinGasLimit
	}
	if gasLimit > constant.MaxGasLimit {
		gasLimit = constant.MaxGasLimit
	}
	w.log.Info("SendTx gasPrice", "gasPrice", gasPrice, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap, "gasLimit", gasLimit,
		"limitMultiplier", w.cfg.LimitMultiplier, "gasMultiplier", w.cfg.GasMultiplier, "nonce", nonce.Uint64())
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
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Data:      input,
		}
	}

	tx := types.NewTx(td)
	chainID := big.NewInt(int64(w.cfg.Id))
	privateKey := w.conn.Keypair().PrivateKey

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
	if err == nil || err.Error() == constant.ErrNonceTooLow.Error() || strings.Index(err.Error(), "nonce too low") != -1 {
		return true
	}

	return false
}
