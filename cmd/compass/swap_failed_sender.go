package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/ChainSafe/log15"
	ethereum "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	tronaddress "github.com/lbtsm/gotron-sdk/pkg/address"
	gtronclient "github.com/lbtsm/gotron-sdk/pkg/client"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/internal/constant"
	cpkeystore "github.com/mapprotocol/compass/pkg/keystore"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// minGasPriceByChain enforces a floor for chains where validators reject very
// low gas prices regardless of mempool fullness. Keys are decimal chain ids.
// Values are in wei (1 gwei = 1e9). Keep this conservative — most chains'
// suggested gasPrice from a healthy RPC is already sufficient; this map is
// only a safety net against buggy RPCs returning sub-minimum quotes.
var minGasPriceByChain = map[string]*big.Int{
	"1":     big.NewInt(1_000_000_000),  // ETH mainnet — 1 gwei
	"137":   big.NewInt(30_000_000_000), // Polygon — 30 gwei (post-MaticV2 floor)
	"22776": big.NewInt(500_000_000),    // MAP — 0.5 gwei
	// BSC accepts ≤0.05 gwei in practice; no floor here. If your RPC quotes
	// absurdly low, switch to a public broadcast endpoint instead of forcing.
}

// errIgnorable wraps an underlying error whose message matched one of
// constant.IgnoreError's substrings (e.g. "order exist", "already verified").
// Callers should treat this as success: rescue isn't needed because the
// bridge has already settled the order on its own.
var errIgnorable = errors.New("ignorable")

// matchIgnore returns the first IgnoreError substring contained in msg, or ""
// if none matches.
func matchIgnore(msg string) string {
	for pat := range constant.IgnoreError {
		if pat != "" && strings.Contains(msg, pat) {
			return pat
		}
	}
	return ""
}

// senderRegistry caches per-chain ethclient connections and lazily resolves
// the right sender (EVM vs Tron) for whatever chainId butter-api returns.
// The same secp256k1 private key signs on every chain: tron addresses are
// derived from the same eth-style public key (prefix 0x → 0x41).
type senderRegistry struct {
	mu sync.Mutex

	// Shared key (secp256k1). EVM uses Address directly; Tron uses tronFrom.
	evmKey   *ecdsa.PrivateKey
	evmFrom  ethcommon.Address
	tronFrom string // base58 T..., derived from evmFrom

	// EVM
	evmEndpoints map[string]string // chainId(decimal string) → endpoint
	evmClients   map[string]*ethclient.Client

	// Tron
	tronEndpoint string
	tronClient   *gtronclient.GrpcClient
}

// chainNonNativeID returns the decimal-string form of a numeric chain id.
func chainNonNativeID(s string) string {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return strconv.FormatInt(n, 10)
	}
	return s
}

func newSenderRegistry(cfg *config.Config) (*senderRegistry, error) {
	if cfg.Other.SwapFailedKeystore == "" {
		return nil, fmt.Errorf("config.other.swap_failed_keystore is required")
	}
	kp, err := cpkeystore.KeypairFromEth(cfg.Other.SwapFailedKeystore)
	if err != nil {
		return nil, fmt.Errorf("load swap_failed_keystore: %w", err)
	}

	// Same secp256k1 key signs both EVM and Tron txs. Tron address is just
	// the eth address with the "0x" replaced by "41" + base58check.
	tronHex := "41" + strings.TrimPrefix(strings.ToLower(kp.Address.Hex()), "0x")
	tronFrom := tronaddress.HexToAddress(tronHex).String()

	r := &senderRegistry{
		evmKey:       kp.PrivateKey,
		evmFrom:      kp.Address,
		tronFrom:     tronFrom,
		evmEndpoints: make(map[string]string),
		evmClients:   make(map[string]*ethclient.Client),
	}

	tronChainID := strconv.FormatInt(constant.TronChainId, 10)
	for _, c := range cfg.Chains {
		id := chainNonNativeID(c.Id)
		switch strings.ToLower(c.Type) {
		case constant.Tron:
			if id == tronChainID {
				r.tronEndpoint = c.Endpoint
			}
		default:
			// Treat any non-tron entry as EVM. Non-EVM chains (sol/ton/btc) we
			// don't sign for; rescue tx for those just won't have an endpoint
			// and will get skipped with an error at send time.
			r.evmEndpoints[id] = c.Endpoint
		}
	}
	// Map chain is in cfg.MapChain, not cfg.Chains.
	if cfg.MapChain.Endpoint != "" {
		r.evmEndpoints[chainNonNativeID(cfg.MapChain.Id)] = cfg.MapChain.Endpoint
	}
	return r, nil
}

// send routes the tx to the correct underlying sender based on chain id.
// Returns the tx hash on success.
func (r *senderRegistry) send(tx txParam, logger log.Logger) (string, error) {
	chainID := chainNonNativeID(tx.ChainID)
	if chainID == strconv.FormatInt(constant.TronChainId, 10) {
		return r.sendTron(tx, logger)
	}
	// Refuse non-EVM chains we explicitly don't handle yet. fex-web has
	// separate flows for solana/ton/btc; until those are wired up here we
	// surface a clear error so the alarm tells the operator to act manually.
	switch chainID {
	case strconv.FormatInt(constant.TonChainId, 10):
		return "", fmt.Errorf("ton chain rescue not supported (chainId=%s)", chainID)
	case strconv.FormatInt(constant.BtcChainId, 10):
		return "", fmt.Errorf("btc chain rescue not supported (chainId=%s)", chainID)
	case strconv.FormatInt(constant.SolMainChainId, 10),
		strconv.FormatInt(constant.SolTestChainId, 10):
		return "", fmt.Errorf("solana chain rescue not supported (chainId=%s)", chainID)
	}
	return r.sendEvm(tx, logger)
}

func (r *senderRegistry) sendEvm(tx txParam, logger log.Logger) (string, error) {
	chainID := chainNonNativeID(tx.ChainID)
	endpoint, ok := r.evmEndpoints[chainID]
	if !ok {
		return "", fmt.Errorf("no rpc endpoint configured for evm chainId=%s", chainID)
	}

	client, err := r.evmClient(chainID, endpoint)
	if err != nil {
		return "", err
	}

	value, ok := parseBigDec(tx.Value)
	if !ok {
		return "", fmt.Errorf("parse value: %q", tx.Value)
	}
	data, err := decodeHex(tx.Data)
	if err != nil {
		return "", fmt.Errorf("parse data: %w", err)
	}
	to := ethcommon.HexToAddress(tx.To)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:  r.evmFrom,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		explained := explainEvmCallError(ctx, client, err, endpoint, chainID, r.evmFrom, to, value, data)
		if pat := matchIgnore(explained); pat != "" {
			return "", fmt.Errorf("%w (matched %q): %s", errIgnorable, pat, explained)
		}
		return "", fmt.Errorf("estimate gas: %s", explained)
	}
	gasEstimated := gasLimit
	// fex-web NormalButton uses gas * 1.5; match that.
	gasLimit = gasLimit * 15 / 10
	logger.Info("evm send: gas estimated", "estimated", gasEstimated, "withBuffer", gasLimit)

	nonce, err := client.PendingNonceAt(ctx, r.evmFrom)
	if err != nil {
		return "", fmt.Errorf("pending nonce: %w", err)
	}
	logger.Info("evm send: nonce fetched", "nonce", nonce)

	rawGasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas price: %w", err)
	}
	// Some RPCs return absurdly low values (we've seen 0.05 gwei on BSC) that
	// fall below the chain's validator-enforced minimum, so the tx is accepted
	// by the broadcast endpoint but silently dropped from mempool. Bump 1.2x
	// for headroom and clamp to a per-chain floor.
	gasPrice := new(big.Int).Div(new(big.Int).Mul(rawGasPrice, big.NewInt(12)), big.NewInt(10))
	logger.Info("evm send: gas price",
		"suggested", rawGasPrice.String(), "effective", gasPrice.String())

	chainIDBig, _ := new(big.Int).SetString(chainID, 10)
	rawTx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	signedTx, err := types.SignTx(rawTx, types.NewEIP155Signer(chainIDBig), r.evmKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	logger.Info("evm send: signed",
		"hash", signedTx.Hash().Hex(), "nonce", nonce,
		"gasLimit", gasLimit, "gasPrice", gasPrice.String())

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}
	logger.Info("evm send: broadcast ok", "hash", signedTx.Hash().Hex())
	return signedTx.Hash().Hex(), nil
}

func (r *senderRegistry) evmClient(chainID, endpoint string) (*ethclient.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.evmClients[chainID]; ok {
		return c, nil
	}
	c, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", endpoint, err)
	}
	r.evmClients[chainID] = c
	return c, nil
}

// sendTron signs and broadcasts a tron tx using the same secp256k1 key that
// signs EVM txs. We replicate gotron-sdk's keystore.SignTx inline (sha256 over
// proto-marshaled RawData → crypto.Sign) so the keeper doesn't need a separate
// tron keystore file.
func (r *senderRegistry) sendTron(tx txParam, logger log.Logger) (string, error) {
	if r.tronEndpoint == "" {
		return "", fmt.Errorf("no tron endpoint in cfg.chains")
	}

	cli, err := r.tronGrpc()
	if err != nil {
		return "", err
	}
	logger.Info("tron send: grpc ok", "endpoint", r.tronEndpoint, "from", r.tronFrom)

	to, err := normalizeTronAddress(tx.To)
	if err != nil {
		return "", fmt.Errorf("parse to: %w", err)
	}
	value, ok := parseInt64(tx.Value)
	if !ok {
		return "", fmt.Errorf("parse value: %q", tx.Value)
	}
	input, err := decodeHex(tx.Data)
	if err != nil {
		return "", fmt.Errorf("parse data: %w", err)
	}
	logger.Info("tron send: estimating energy",
		"to", to, "value", value, "dataLen", len(input))

	estimate, err := cli.TriggerConstantContractByEstimate(r.tronFrom, to, input, value)
	if err != nil {
		return "", fmt.Errorf("estimate energy: %w", err)
	}
	for _, v := range estimate.ConstantResult {
		ele := strings.TrimSpace(string(v))
		if ele == "" || ele == "4,^" || ele == "0\xef" {
			continue
		}
		if pat := matchIgnore(ele); pat != "" {
			return "", fmt.Errorf("%w (matched %q): pre-exec revert: %s", errIgnorable, pat, ele)
		}
		return "", fmt.Errorf("pre-exec revert: %s", ele)
	}
	// match chains/tron/writer.go's feeLimit formula: energy * 420 (sun per energy).
	feeLimit := estimate.EnergyUsed * 420
	if feeLimit < 30_000_000 {
		feeLimit = 30_000_000
	}
	logger.Info("tron send: energy estimated",
		"energyUsed", estimate.EnergyUsed, "feeLimit_sun", feeLimit)

	builtTx, err := cli.TriggerContract(r.tronFrom, to, input, feeLimit, value, "", 0)
	if err != nil {
		return "", fmt.Errorf("trigger contract: %w", err)
	}
	logger.Info("tron send: tx built", "txid", hex.EncodeToString(builtTx.GetTxid()))

	rawData, err := proto.Marshal(builtTx.Transaction.GetRawData())
	if err != nil {
		return "", fmt.Errorf("marshal raw: %w", err)
	}
	h := sha256.Sum256(rawData)
	sig, err := ethcrypto.Sign(h[:], r.evmKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	builtTx.Transaction.Signature = append(builtTx.Transaction.Signature, sig)
	logger.Info("tron send: signed", "sigLen", len(sig))

	result, err := cli.Broadcast(builtTx.Transaction)
	if err != nil {
		return "", fmt.Errorf("broadcast: %w", err)
	}
	if !result.GetResult() {
		return "", fmt.Errorf("broadcast rejected: code=%s msg=%s",
			result.GetCode().String(), string(result.GetMessage()))
	}
	txid := hex.EncodeToString(builtTx.GetTxid())
	logger.Info("tron send: broadcast ok", "txid", txid)
	return txid, nil
}

func (r *senderRegistry) tronGrpc() (*gtronclient.GrpcClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tronClient != nil {
		return r.tronClient, nil
	}
	c := gtronclient.NewGrpcClient(r.tronEndpoint)
	if err := c.Start(grpc.WithInsecure()); err != nil {
		return nil, fmt.Errorf("tron grpc start: %w", err)
	}
	r.tronClient = c
	return c, nil
}

// normalizeTronAddress accepts base58 (T...), hex with 0x prefix (0x41...),
// or hex without prefix (41...) and returns the canonical base58 form
// expected by gotron-sdk's TriggerContract.
func normalizeTronAddress(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	// Base58 tron addresses start with "T" and are 34 chars.
	if strings.HasPrefix(s, "T") && len(s) == tronaddress.AddressLengthBase58 {
		return s, nil
	}
	hexStr := strings.TrimPrefix(s, "0x")
	if !strings.HasPrefix(hexStr, "41") {
		// fall back: maybe it's an EVM-style address; tron's gotron-sdk also
		// accepts ETH-style addresses via HexToAddress with a "41" prefix.
		hexStr = "41" + hexStr
	}
	addr := tronaddress.HexToAddress(hexStr)
	if len(addr) == 0 {
		return "", fmt.Errorf("invalid tron addr: %q", s)
	}
	return addr.String(), nil
}

func parseBigDec(s string) (*big.Int, bool) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" || s == "0" {
		return big.NewInt(0), true
	}
	return new(big.Int).SetString(s, 10)
}

func parseInt64(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// explainEvmCallError unwraps a JSON-RPC EstimateGas / Call error so the message
// surfaces a revert reason (decoded if it's the standard Error(string)),
// flags the empty-bytecode case, and prints enough context to replay the call
// elsewhere (cast, tenderly, foundry).
func explainEvmCallError(
	ctx context.Context,
	client *ethclient.Client,
	cause error,
	endpoint, chainID string,
	from, to ethcommon.Address,
	value *big.Int,
	data []byte,
) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%v", cause)

	// Revert data, if the RPC included it.
	if reason, raw, ok := extractRevertReason(cause); ok {
		if reason != "" {
			fmt.Fprintf(&sb, " | revert: %s", reason)
		}
		if raw != "" {
			fmt.Fprintf(&sb, " | revert_data: %s", raw)
		}
	}

	// Common root cause: the to-address has no contract bytecode. EVM treats a
	// call to an EOA with non-empty data as a no-op, but EstimateGas can flag
	// it as a revert depending on the node.
	if code, err := client.CodeAt(ctx, to, nil); err == nil && len(code) == 0 {
		fmt.Fprintf(&sb, " | NO BYTECODE at to=%s (EOA or wrong chain)", to.Hex())
	}

	dataPrefix := ""
	if len(data) >= 4 {
		dataPrefix = "0x" + hex.EncodeToString(data[:4])
	}
	fmt.Fprintf(&sb, " | chain=%s endpoint=%s from=%s to=%s value=%s data_len=%d selector=%s",
		chainID, endpoint, from.Hex(), to.Hex(), value.String(), len(data), dataPrefix)
	return sb.String()
}

// extractRevertReason pulls the JSON-RPC "data" field from a go-ethereum
// EstimateGas / Call error. The data is usually ABI-encoded `Error(string)`
// (selector 0x08c379a0), in which case we decode it; otherwise we return the
// hex blob so the caller can inspect / replay.
func extractRevertReason(err error) (string, string, bool) {
	type dataError interface {
		ErrorData() interface{}
	}
	de, ok := err.(dataError)
	if !ok {
		return "", "", false
	}
	v := de.ErrorData()
	if v == nil {
		return "", "", false
	}
	raw, ok := v.(string)
	if !ok {
		return "", "", false
	}
	if raw == "" || raw == "0x" {
		return "", raw, true
	}
	b, decErr := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if decErr != nil {
		return "", raw, true
	}
	// Error(string) selector + ABI-encoded string.
	if len(b) >= 4+32+32 && b[0] == 0x08 && b[1] == 0xc3 && b[2] == 0x79 && b[3] == 0xa0 {
		// offset is at b[4:36] (always 0x20), length at b[36:68], data at b[68:68+len]
		strLen := new(big.Int).SetBytes(b[36:68]).Int64()
		end := int64(68) + strLen
		if end <= int64(len(b)) {
			return string(b[68:end]), raw, true
		}
	}
	return "", raw, true
}
