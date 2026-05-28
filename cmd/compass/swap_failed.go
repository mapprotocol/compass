package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/config"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/pkg/util"
	"github.com/urfave/cli/v2"
)

// BRIDGE_STATE enum (matches butter-api TypeScript definition).
const (
	stateSourcePending   = 0
	stateSourceConfirmed = 1
	stateRelayPending    = 2
	stateRelayConfirmed  = 3
	stateRelayRetry      = 4
	stateDestPending     = 5
	stateDestSwapPending = 6
	stateDestSwapRescue  = 7
	stateDestConfirmed   = 8
	stateSourceFailed    = 9
	stateRelayFailed     = 10
	stateDestFailed      = 11
	stateDestSwapFailed  = 12
)

const (
	// query window: [now-24h, now]
	swapFailedWindow   = 24 * time.Hour
	swapFailedInterval = 1 * time.Minute
	swapFailedPageSize = 100
	// minTxAge is how old a pending tx must be before we touch it. Fresh
	// failures often resolve themselves once the relay or dest swap retries
	// on butter-api's side; jumping in early just races them.
	minTxAge        = 10 * time.Minute
	pendingURL      = "https://opapi.chainservice.io/api/transaction/pendings"
	execURL         = "https://opapi.chainservice.io/api/transaction/exec"
	defaultSlippage = "100" // 1%, matches fex-web Slippages[0]
	maxAttempts     = 4     // 1 initial + 3 retries
)

// retryDelays is the wait after each failed attempt before the next one.
// After the last delay's attempt fails, we alarm and give up.
var retryDelays = []time.Duration{
	1 * time.Minute,
	2 * time.Minute,
	3 * time.Minute,
}

// retryEntry tracks per-tx state across poll cycles.
type retryEntry struct {
	attempts int       // number of attempts that have already finished (success or failure)
	nextAt   time.Time // earliest time to attempt next; zero = ready now
	done     bool      // succeeded or gave up
	lastErr  string
}

var (
	seenFailed   = make(map[string]*retryEntry)
	seenFailedMu sync.Mutex
)

var swapFailedCommand = cli.Command{
	Name:        "swap-failed",
	Usage:       "poll butter api for DEST_SWAP_FAILED transactions and auto-rescue",
	Description: "Periodically queries butter api pending transactions, POSTs /api/transaction/exec for each newly-detected failure, and signs+sends the returned tx on the appropriate chain. Retries 1/2/3 min on failure, alarms after 4 attempts.",
	Action:      swapFailed,
	Flags:       append(app.Flags, cliFlags...),
}

type pendingTx struct {
	ID               string      `json:"id"`
	OrderID          string      `json:"orderId"`
	SourceHash       string      `json:"sourceHash"`
	SourceHeight     string      `json:"sourceHeight"`
	SourceChain      chainInfo   `json:"sourceChain"`
	DestinationHash  string      `json:"destinationHash"`
	DestinationChain chainInfo   `json:"destinationChain"`
	RelayHash        string      `json:"relayHash"`
	RelayInHash      string      `json:"relayInHash"`
	RelayHeight      string      `json:"relayHeight"`
	State            int         `json:"state"`
	SourceState      int         `json:"sourceState"`
	RelayState       int         `json:"relayState"`
	DestinationState int         `json:"destinationState"`
	Sender           string      `json:"sender"`
	Receiver         string      `json:"receiver"`
	SendTime         string      `json:"sendTime"` // RFC3339 e.g. "2026-05-15T02:47:11.000Z"
	Logs             []txLog     `json:"logs"`
	Affiliates       []affiliate `json:"affiliates"`
}

type txLog struct {
	ChainID  string `json:"chainId"`
	LogIndex int64  `json:"logIndex"`
}

type affiliate struct {
	Name string `json:"name"`
}

type chainInfo struct {
	ChainID int64  `json:"chainId"`
	Name    string `json:"name"`
}

type pendingResponse struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    struct {
		Total int         `json:"total"`
		Pages int         `json:"pages"`
		Page  int         `json:"page"`
		Size  int         `json:"size"`
		Items []pendingTx `json:"items"`
	} `json:"data"`
}

// proofParams mirrors fex-web's handleExecParams body (DetailModal/index.tsx L788-808).
type proofParams struct {
	SrcChain         string `json:"src_chain"`
	SrcTxHash        string `json:"src_tx_hash"`
	SrcLogIndex      int64  `json:"src_log_index"`
	SrcBlockNumber   int64  `json:"src_block_number"`
	RelayChain       string `json:"relay_chain"`
	RelayTxHash      string `json:"relay_tx_hash"`
	RelayLogIndex    int64  `json:"relay_log_index"`
	RelayBlockNumber int64  `json:"relay_block_number"`
	Status           int    `json:"status"`
	DesChain         string `json:"des_chain"`
	DesTxHash        string `json:"des_tx_hash"`
	DesLogIndex      int64  `json:"des_log_index"`
	Slippage         string `json:"slippage"`
	Entrance         string `json:"entrance"`
}

// execResponse parses /api/transaction/exec. Three different shapes share one struct;
// pickTx dispatches based on UserRouter + ExecRelay.
type execResponse struct {
	Errno   int       `json:"errno"`
	Message string    `json:"message"`
	Data    *execData `json:"data"`
}

type execData struct {
	ExecChain  string     `json:"execChain"`
	ExecData   string     `json:"execData"`
	ExecDesc   string     `json:"execDesc"`
	ExecTo     string     `json:"execTo"`
	ExecRelay  bool       `json:"execRelay"`
	UserRouter bool       `json:"userRouter"`
	ExecRoute  *execRoute `json:"execRoute"`
}

type execRoute struct {
	MinReceivedInLog   string        `json:"minReceivedInLog"`
	RescueFundsTxParam *txParam      `json:"rescueFundsTxParam"`
	TxParam            []txParam     `json:"txParam"`           // execRelay=true shape
	RouteWithTxParams  []routeWithTx `json:"routeWithTxParams"` // rescue shape
}

type routeWithTx struct {
	TxParam []txParam `json:"txParam"`
}

type txParam struct {
	To      string `json:"to"`
	Data    string `json:"data"`
	Value   string `json:"value"`
	ChainID string `json:"chainId"`
	Method  string `json:"method"`
}

func swapFailed(ctx *cli.Context) error {
	if err := startLogger(ctx); err != nil {
		return err
	}
	log.Info("Starting swap-failed poller...")

	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	util.Init(cfg.Other.Env, cfg.Other.MonitorUrl)

	// --keystore on the command line overrides cfg.Other.SwapFailedKeystore so
	// the keeper can use a different keystore from the relayer's chain configs.
	// Use IsSet (not String) because the flag has a default value of "./keys",
	// which would otherwise silently clobber the config every time.
	if ctx.IsSet(config.KeystorePathFlag.Name) {
		cfg.Other.SwapFailedKeystore = ctx.String(config.KeystorePathFlag.Name)
	}

	sender, err := newSenderRegistry(cfg)
	if err != nil {
		return fmt.Errorf("init senders: %w", err)
	}

	httpc := &http.Client{Timeout: 30 * time.Second}

	// run once immediately, then on ticker
	pollOnce(httpc, sender)
	ticker := time.NewTicker(swapFailedInterval)
	defer ticker.Stop()
	for range ticker.C {
		pollOnce(httpc, sender)
	}
	return nil
}

func pollOnce(httpc *http.Client, sender *senderRegistry) {
	now := time.Now()
	end := now
	start := end.Add(-swapFailedWindow)
	endMs := end.UnixMilli()
	startMs := start.UnixMilli()

	page := 1
	failedCount, scheduledCount, attemptedCount := 0, 0, 0
	for {
		items, pages, err := fetchPending(httpc, page, swapFailedPageSize, startMs, endMs)
		if err != nil {
			log.Error("fetch pending failed", "page", page, "err", err)
			return
		}
		for _, tx := range items {
			failedCount++
			// Skip txs younger than minTxAge so we don't race butter-api's own
			// retry pipeline. SendTime is RFC3339 (e.g. "2026-05-15T02:47:11.000Z").
			if t, ok := parseSendTime(tx.SendTime); ok && now.Sub(t) < minTxAge {
				continue
			}
			if !shouldAttempt(tx.ID, now) {
				continue
			}
			scheduledCount++
			attempt(httpc, sender, tx)
			attemptedCount++
		}
		if page >= pages || pages == 0 {
			break
		}
		page++
	}
	evictSeen()
	log.Info("poll cycle done",
		"start", startMs, "end", endMs,
		"dest_swap_failed", failedCount,
		"due", scheduledCount,
		"attempted", attemptedCount)
}

// shouldAttempt decides whether tx.ID is due for the next retry. Always returns
// true for first sighting; on subsequent polls only when the back-off window has
// passed and we haven't yet given up.
func shouldAttempt(id string, now time.Time) bool {
	seenFailedMu.Lock()
	defer seenFailedMu.Unlock()
	e, ok := seenFailed[id]
	if !ok {
		seenFailed[id] = &retryEntry{}
		return true
	}
	if e.done {
		return false
	}
	return !now.Before(e.nextAt)
}

// attempt runs one rescue attempt: POST /api/transaction/exec → pick tx → sign+send.
// Records outcome in seenFailed and decides whether to re-arm or alarm.
func attempt(httpc *http.Client, sender *senderRegistry, tx pendingTx) {
	params := buildProofParams(tx)
	logger := log.New("orderId", tx.OrderID, "destChain", tx.DestinationChain.Name)

	logger.Info("rescue attempt start",
		"state", tx.State, "srcChain", tx.SourceChain.Name,
		"srcHash", tx.SourceHash, "destHash", tx.DestinationHash,
		"sendTime", tx.SendTime)
	data, err := fetchExecData(httpc, params)
	if err != nil {
		recordAttempt(tx.ID, tx.OrderID, fmt.Errorf("fetch exec data: %w", err), logger)
		return
	}
	logger.Info("rescue exec data received",
		"execChain", data.ExecChain, "execTo", data.ExecTo,
		"execDesc", data.ExecDesc, "execDataLen", len(data.ExecData),
		"userRouter", data.UserRouter, "execRelay", data.ExecRelay,
		"hasExecRoute", data.ExecRoute != nil,
		"hasRescueFunds", data.ExecRoute != nil && data.ExecRoute.RescueFundsTxParam != nil,
		"routeWithTxParamsLen", routeWithTxParamsLen(data),
		"execRouteTxParamLen", execRouteTxParamLen(data))

	chosen, err := pickTx(data)
	if err != nil {
		recordAttempt(tx.ID, tx.OrderID, fmt.Errorf("pick tx: %w", err), logger)
		return
	}
	logger.Info("rescue tx picked",
		"chainId", chosen.ChainID, "to", chosen.To, "method", chosen.Method)

	hash, err := sender.send(*chosen, logger)
	if err != nil {
		// Estimate / pre-exec matched constant.IgnoreError → bridge already
		// settled this order, no rescue needed. Mark done so we don't retry.
		if errors.Is(err, errIgnorable) {
			markDone(tx.ID)
			logger.Info("rescue skipped (ignorable)",
				"chainId", chosen.ChainID, "to", chosen.To, "reason", err)
			return
		}
		recordAttempt(tx.ID, tx.OrderID, fmt.Errorf("send tx: %w", err), logger)
		return
	}
	markDone(tx.ID)
	logger.Info("rescue tx sent",
		"chainId", chosen.ChainID, "to", chosen.To,
		"hash", hash, "desc", data.ExecDesc)
}

func routeWithTxParamsLen(d *execData) int {
	if d == nil || d.ExecRoute == nil {
		return 0
	}
	return len(d.ExecRoute.RouteWithTxParams)
}

func execRouteTxParamLen(d *execData) int {
	if d == nil || d.ExecRoute == nil {
		return 0
	}
	return len(d.ExecRoute.TxParam)
}

func txSelector(hexData string) string {
	s := strings.TrimPrefix(hexData, "0x")
	if len(s) < 8 {
		return ""
	}
	return "0x" + s[:8]
}

func recordAttempt(id, orderID string, err error, logger log.Logger) {
	seenFailedMu.Lock()
	e := seenFailed[id]
	if e == nil {
		e = &retryEntry{}
		seenFailed[id] = e
	}
	e.attempts++
	e.lastErr = err.Error()
	logger.Warn("rescue attempt failed", "attempt", e.attempts, "max", maxAttempts, "err", err)
	if e.attempts >= maxAttempts {
		e.done = true
		seenFailedMu.Unlock()
		util.Alarm(context.Background(),
			fmt.Sprintf("swap-failed rescue gave up after %d attempts, orderId=%s lastErr=%s",
				e.attempts, orderID, err.Error()))
		return
	}
	delay := retryDelays[e.attempts-1]
	e.nextAt = time.Now().Add(delay)
	seenFailedMu.Unlock()
	logger.Info("rescue scheduled for retry", "wait", delay)
}

func markDone(id string) {
	seenFailedMu.Lock()
	defer seenFailedMu.Unlock()
	e := seenFailed[id]
	if e == nil {
		e = &retryEntry{}
		seenFailed[id] = e
	}
	e.done = true
}

// evictSeen drops cache entries that are done and older than 24h. Pending
// (not-yet-done) entries are kept regardless so their retry schedule survives.
func evictSeen() {
	cutoff := time.Now().Add(-24 * time.Hour)
	seenFailedMu.Lock()
	defer seenFailedMu.Unlock()
	for id, e := range seenFailed {
		if e.done && e.nextAt.Before(cutoff) {
			delete(seenFailed, id)
		}
	}
}

func isDestSwapFailed(tx pendingTx) bool {
	return tx.State == stateDestSwapFailed
}

func fetchPending(httpc *http.Client, page, size int, startMs, endMs int64) ([]pendingTx, int, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	q.Set("start", strconv.FormatInt(startMs, 10))
	q.Set("end", strconv.FormatInt(endMs, 10))
	reqURL := pendingURL + "?" + q.Encode()

	resp, err := httpc.Get(reqURL)
	if err != nil {
		return nil, 0, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}
	var pr pendingResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, 0, fmt.Errorf("unmarshal: %w", err)
	}
	if pr.Errno != 0 {
		return nil, 0, fmt.Errorf("api errno=%d msg=%s", pr.Errno, pr.Message)
	}
	return pr.Data.Items, pr.Data.Pages, nil
}

// fetchExecData POSTs proofParams and returns the parsed exec data shape.
func fetchExecData(httpc *http.Client, params proofParams) (*execData, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, execURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d body %s", resp.StatusCode, string(b))
	}
	var er execResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if er.Errno != 0 {
		return nil, fmt.Errorf("api errno=%d msg=%s", er.Errno, er.Message)
	}
	if er.Data == nil {
		return nil, fmt.Errorf("empty data")
	}
	return er.Data, nil
}

// pickTx selects the tx to send:
//   - userRouter=false → use top-level execData/execTo/execChain directly
//   - userRouter=true  → use execRoute.rescueFundsTxParam (its own to/data/value/chainId)
func pickTx(d *execData) (*txParam, error) {
	if d == nil {
		return nil, fmt.Errorf("nil exec data")
	}
	if !d.UserRouter {
		return &txParam{
			To:      d.ExecTo,
			Data:    d.ExecData,
			Value:   "0",
			ChainID: d.ExecChain,
		}, nil
	}
	if d.ExecRoute == nil || d.ExecRoute.RescueFundsTxParam == nil {
		return nil, fmt.Errorf("userRouter=true but no execRoute.rescueFundsTxParam")
	}
	return d.ExecRoute.RescueFundsTxParam, nil
}

// buildProofParams mirrors fex-web's DetailModal handleExecParams (L755-810).
func buildProofParams(tx pendingTx) proofParams {
	srcChainStr := strconv.FormatInt(tx.SourceChain.ChainID, 10)
	desChainStr := strconv.FormatInt(tx.DestinationChain.ChainID, 10)
	srcLogIdx := pickLogIndex(tx.Logs, srcChainStr, false /*max*/)
	relayLogIdx := pickLogIndex(tx.Logs, "22776", true)
	destLogIdx := pickLogIndex(tx.Logs, desChainStr, true)

	relayHash := tx.RelayHash
	if relayHash == "" {
		relayHash = tx.RelayInHash
	}
	entrance := ""
	if len(tx.Affiliates) > 0 {
		entrance = tx.Affiliates[0].Name
	}

	return proofParams{
		SrcChain:         srcChainStr,
		SrcTxHash:        normalizeHash(tx.SourceHash, srcChainStr),
		SrcLogIndex:      srcLogIdx,
		SrcBlockNumber:   atoiSafe(tx.SourceHeight),
		RelayChain:       "22776",
		RelayTxHash:      normalizeHash(relayHash, "22776"),
		RelayLogIndex:    relayLogIdx,
		RelayBlockNumber: atoiSafe(tx.RelayHeight),
		Status:           getStatus(tx),
		DesChain:         desChainStr,
		DesTxHash:        normalizeHash(tx.DestinationHash, desChainStr),
		DesLogIndex:      destLogIdx,
		Slippage:         defaultSlippage,
		Entrance:         entrance,
	}
}

// pickLogIndex returns either the lowest (max=false) or highest (max=true)
// logIndex among logs on the given chain. fex-web uses min for src logs
// (ascending sort, take [0]) and max for relay/dest logs (descending sort,
// take [0]).
func pickLogIndex(logs []txLog, chainID string, max bool) int64 {
	var chosen int64
	found := false
	for _, l := range logs {
		if l.ChainID != chainID {
			continue
		}
		if !found {
			chosen = l.LogIndex
			found = true
			continue
		}
		if max && l.LogIndex > chosen {
			chosen = l.LogIndex
		} else if !max && l.LogIndex < chosen {
			chosen = l.LogIndex
		}
	}
	return chosen
}

// getStatus mirrors fex-web DetailModal getStatus (L705-740).
func getStatus(tx pendingTx) int {
	result := 0
	switch tx.State {
	case stateDestConfirmed, stateDestSwapRescue:
		result = 1
	case stateSourceFailed:
		result = 0
	case stateDestFailed, stateDestSwapFailed:
		result = 4
	case stateSourcePending:
		result = 0
	case stateRelayConfirmed:
		result = 3
	case stateRelayFailed, stateRelayRetry:
		result = 2
	}
	if result == 0 {
		if tx.RelayInHash != "" {
			return 2
		}
		if tx.RelayHash != "" {
			return 3
		}
	}
	return result
}

// normalizeHash mirrors fex-web getHash: tron tx hashes need a 0x prefix
// before being sent in proof params.
func normalizeHash(hash, chainID string) string {
	if hash == "" {
		return ""
	}
	if chainID == strconv.FormatInt(constant.TronChainId, 10) && !strings.HasPrefix(hash, "0x") {
		return "0x" + hash
	}
	return hash
}

func atoiSafe(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// parseSendTime parses the RFC3339 timestamp butter-api uses for sendTime.
func parseSendTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
