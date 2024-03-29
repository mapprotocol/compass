// Copyright 2019 Conflux Foundation. All rights reserved.
// Conflux is free software and distributed under GNU General Public License.
// See http://www.gnu.org/licenses/

package conflux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/mapprotocol/compass/internal/conflux/types"
	"github.com/pkg/errors"
)

const (
	contentType = "application/json"
	vsn         = "2.0"
)

var ErrNoResult = errors.New("no result in JSON-RPC response")

type Client struct {
	client    *http.Client
	url       string
	closeOnce sync.Once
	closch    chan interface{}
	mu        sync.Mutex // protcts headers
	headers   http.Header
	isHttp    bool
	idCounter uint32
}

func NewClient(endpoint string) (*Client, error) {
	// Sanity chck URL so we don't end up with a client that will fail every request.
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header, 2)
	headers.Set("accept", contentType)
	headers.Set("content-type", contentType)
	return &Client{
		client:    new(http.Client),
		url:       endpoint,
		closeOnce: sync.Once{},
		closch:    make(chan interface{}),
		mu:        sync.Mutex{},
		headers:   headers,
	}, nil
}

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	data, err := c.call(ctx, "pos_getStatus")
	if err != nil {
		return nil, err
	}
	var ret Status
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetBlockByNumber(ctx context.Context, blockNumber BlockNumber) (*Block, error) {
	data, err := c.call(ctx, "pos_getBlockByNumber", blockNumber)
	if err != nil {
		return nil, err
	}
	var ret Block
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetLedgerInfoByEpochAndRound(ctx context.Context, epochNumber hexutil.Uint64,
	round hexutil.Uint64) (*LedgerInfoWithSignatures, error) {
	data, err := c.call(ctx, "pos_getLedgerInfoByEpochAndRound", epochNumber, round)
	if err != nil {
		return nil, err
	}
	var ret LedgerInfoWithSignatures
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetBlockByEpochNumber(ctx context.Context, blockNumber hexutil.Uint64) (*BlockSummary, error) {
	data, err := c.call(ctx, "cfx_getBlockByEpochNumber", blockNumber, false)
	if err != nil {
		return nil, err
	}

	var ret BlockSummary
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetEpochReceipts(ctx context.Context, epoch types.EpochOrBlockHash,
	includeEthReceipts ...bool) ([][]types.TransactionReceipt, error) {
	includeEth := get1stBoolIfy(includeEthReceipts)
	data, err := c.call(ctx, "cfx_getEpochReceipts", epoch, includeEth)
	if err != nil {
		return nil, err
	}

	var ret [][]types.TransactionReceipt
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func get1stBoolIfy(values []bool) bool {
	value := false
	if len(values) > 0 {
		value = values[0]
	}
	return value
}

func (c *Client) call(ctx context.Context, method string, args ...interface{}) ([]byte, error) {
	var raw json.RawMessage
	err := c.CallContext(ctx, &raw, method, args...)
	if err != nil {
		return nil, err
	} else if len(raw) == 0 {
		return nil, ethereum.NotFound
	}

	data, err := raw.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return data, nil
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (err *jsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

type requestOp struct {
	ids  []json.RawMessage
	err  error
	resp chan *jsonrpcMessage // receives up to len(ids) responses
}

func (op *requestOp) wait(ctx context.Context) (*jsonrpcMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if result != nil && reflect.TypeOf(result).Kind() != reflect.Ptr {
		return fmt.Errorf("call result parameter must be pointer or nil interface: %v", result)
	}
	msg, err := c.newMessage(method, args...)
	if err != nil {
		return err
	}
	op := &requestOp{ids: []json.RawMessage{msg.ID}, resp: make(chan *jsonrpcMessage, 1)}

	err = c.sendHTTP(ctx, op, msg)
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel when it quits.
	switch resp, err := op.wait(ctx); {
	case err != nil:
		return err
	case resp.Error != nil:
		return resp.Error
	case len(resp.Result) == 0:
		return ErrNoResult
	default:
		return json.Unmarshal(resp.Result, &result)
	}
}

func (c *Client) nextID() json.RawMessage {
	id := atomic.AddUint32(&c.idCounter, 1)
	return strconv.AppendUint(nil, uint64(id), 10)
}

func (c *Client) newMessage(method string, paramsIn ...interface{}) (*jsonrpcMessage, error) {
	msg := &jsonrpcMessage{Version: vsn, ID: c.nextID(), Method: method}
	if paramsIn != nil { // prevent sending "params":null
		var err error
		if msg.Params, err = json.Marshal(paramsIn); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (c *Client) sendHTTP(ctx context.Context, op *requestOp, msg interface{}) error {
	respBody, err := c.doRequest(ctx, msg)
	if err != nil {
		return err
	}
	defer respBody.Close()

	var respmsg jsonrpcMessage
	if err := json.NewDecoder(respBody).Decode(&respmsg); err != nil {
		return err
	}
	op.resp <- &respmsg
	return nil
}

func (c *Client) doRequest(ctx context.Context, msg interface{}) (io.ReadCloser, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, ioutil.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(body))

	// set headers
	c.mu.Lock()
	req.Header = c.headers.Clone()
	c.mu.Unlock()

	// do request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		//var buf bytes.Buffer
		//var body []byte
		//if _, err := buf.ReadFrom(resp.Body); err == nil {
		//	body = buf.Bytes()
		//}

		return nil, fmt.Errorf("conflux request code is(%d)", resp.StatusCode)
		//return nil, HTTPError{
		//	Status:     resp.Status,
		//	StatusCode: resp.StatusCode,
		//	Body:       body,
		//}
	}
	return resp.Body, nil
}
