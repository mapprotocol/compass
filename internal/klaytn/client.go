package klaytn

import (
	"net/http"
	"sync"
)

const (
	contentType = "application/json"
)

type Client struct {
	client    *http.Client
	url       string
	closeOnce sync.Once
	closeCh   chan interface{}
	mu        sync.Mutex // protects headers
	headers   http.Header
}

func DialHttp(endpoint string) (*Client, error) {
	//// Sanity check URL so we don't end up with a client that will fail every request.
	//_, err := url.Parse(endpoint)
	//if err != nil {
	//	return nil, err
	//}
	//
	//initctx := context.Background()
	//headers := make(http.Header, 2)
	//headers.Set("accept", contentType)
	//headers.Set("content-type", contentType)
	return &Client{
		client:    nil,
		url:       "",
		closeOnce: sync.Once{},
		closeCh:   nil,
		mu:        sync.Mutex{},
		headers:   nil,
	}, nil
}
