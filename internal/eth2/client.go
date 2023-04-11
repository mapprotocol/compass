package eth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mapprotocol/compass/internal/constant"
)

const (
	contentType = "application/json"
	vsn         = "2.0"
)

var ErrNoResult = errors.New("no result in JSON-RPC response")

type Client struct {
	client    *http.Client
	endpoint  string
	closeOnce sync.Once
	closch    chan interface{}
	mu        sync.Mutex // protcts headers
	headers   http.Header
	isHttp    bool
	idCounter uint32
}

func DialHttp(endpoint string) (*Client, error) {
	// Sanity chck URL so we don't end up with a client that will fail every request.
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header, 2)
	headers.Set("accept", contentType)
	headers.Set("content-type", contentType)
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	return &Client{
		client:    client,
		endpoint:  endpoint,
		closeOnce: sync.Once{},
		closch:    make(chan interface{}),
		mu:        sync.Mutex{},
		headers:   headers,
	}, nil
}

func (c *Client) BeaconHeaders(ctx context.Context, blockId constant.BlockIdOfEth2) (*BeaconHeadersResp, error) {
	urlPath := fmt.Sprintf("%s/%s/%s", c.endpoint, "eth/v1/beacon/headers", blockId)
	var ret BeaconHeadersResp
	err := c.CallContext(ctx, urlPath, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) LightClientUpdate(ctx context.Context, startPeriod uint64) (*LightClientUpdatesResp, error) {
	urlPath := fmt.Sprintf("%s/%s?start_period=%d&count=1", c.endpoint, "eth/v1/beacon/light_client/updates", startPeriod)
	respBody, err := c.doRequest(ctx, urlPath)
	if err != nil {
		return nil, err
	}
	defer respBody.Close()
	var respMsg []CommonData
	if err = json.NewDecoder(respBody).Decode(&respMsg); err != nil {
		return nil, err
	}
	var ret LightClientUpdatesResp
	data, _ := json.Marshal(respMsg[0])
	err = json.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) FinallyUpdate(ctx context.Context) (*FinalityUpdateResp, error) {
	urlPath := fmt.Sprintf("%s/%s", c.endpoint, "eth/v1/beacon/light_client/finality_update")
	var ret FinalityUpdateResp
	err := c.CallContext(ctx, urlPath, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (c *Client) GetBlocks(ctx context.Context, blockId string) (*BlocksResp, error) {
	urlPath := fmt.Sprintf("%s/%s/%s", c.endpoint, "eth/v2/beacon/blocks", blockId)
	var ret BlocksResp
	err := c.CallContext(ctx, urlPath, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type requestOp struct {
	ids  []json.RawMessage
	err  error
	resp chan *CommonData // receives up to len(ids) responses
}

func (op *requestOp) wait(ctx context.Context) (*CommonData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

func (c *Client) CallContext(ctx context.Context, url string, result interface{}) error {
	if result != nil && reflect.TypeOf(result).Kind() != reflect.Ptr {
		return fmt.Errorf("call result parameter must be pointer or nil interface: %v", result)
	}

	op := &requestOp{ids: []json.RawMessage{c.nextID()}, resp: make(chan *CommonData, 1)}

	err := c.sendHTTP(ctx, url, op)
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel when it quits.
	switch resp, err := op.wait(ctx); {
	case err != nil:
		return err
	case resp.StatusCode == 404:
		return ErrNoResult
	case resp.Error != "":
		return errors.New(resp.Error)
	default:
		data, _ := json.Marshal(resp)
		return json.Unmarshal(data, &result)
	}
}

func (c *Client) nextID() json.RawMessage {
	id := atomic.AddUint32(&c.idCounter, 1)
	return strconv.AppendUint(nil, uint64(id), 10)
}

func (c *Client) sendHTTP(ctx context.Context, url string, op *requestOp) error {
	respBody, err := c.doRequest(ctx, url)
	if err != nil {
		return err
	}
	defer respBody.Close()

	var respMsg CommonData
	if err = json.NewDecoder(respBody).Decode(&respMsg); err != nil {
		return err
	}
	op.resp <- &respMsg
	return nil
}

func (c *Client) doRequest(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("eth2 doRequest failed, code %v", resp.StatusCode)
	}
	return resp.Body, nil
}
