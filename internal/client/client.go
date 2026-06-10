package client

import (
	"bytes"
	"io"
	"net/http"
	"time"

	log "github.com/ChainSafe/log15"
)

var (
	cli = http.Client{
		Timeout: 30 * time.Second,
	}
)

func JsonPost(url string, data []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		log.Info("JsonPost", "url", url, "duration", time.Since(start))
	}()
	resp, err := cli.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Error("JsonPost request error", "url", url, "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("JsonPost io.ReadAll error", "url", url, "error", err)
		return nil, err
	}
	return body, nil
}

func JsonGet(url string) ([]byte, error) {
	return JsonGetWithHeaders(url, nil)
}

func JsonGetWithHeaders(url string, headers map[string]string) ([]byte, error) {
	start := time.Now()
	defer func() {
		log.Info("JsonGet", "url", url, "duration", time.Since(start))
	}()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error("JsonGet request build error", "url", url, "error", err)
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := cli.Do(req)
	if err != nil {
		log.Error("JsonGet request error", "url", url, "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("JsonGet io.ReadAll error", "url", url, "error", err)
		return nil, err
	}
	return body, nil
}
