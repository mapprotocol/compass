package klaytn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func DialHttp(endpoint string, isHttp bool) (*Client, error) {
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

func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*RpcHeader, error) {
	return c.getBlock(ctx, "klay_getBlockByNumber", toBlockNumArg(number), true)
}

func (c *Client) TransactionReceiptRpcOutput(ctx context.Context, txHash common.Hash) (map[string]interface{}, error) {
	return c.getReceipt(ctx, "klay_getTransactionReceipt", txHash)
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func (c *Client) getBlock(ctx context.Context, method string, args ...interface{}) (*RpcHeader, error) {
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

	var ret RpcHeader
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) getReceipt(ctx context.Context, method string, args ...interface{}) (map[string]interface{}, error) {
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

	var ret map[string]interface{}
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
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

		//return nil, nil
		return nil, fmt.Errorf("klaytn request code is(%d)", resp.StatusCode)
		//HTTPError{
		//	Status:     resp.Status,
		//	StatusCode: resp.StatusCode,
		//	Body:       body,
		//}
	}
	return resp.Body, nil
}
