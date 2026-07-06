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
	return JsonPostWithHeaders(url, data, nil)
}

func JsonPostWithHeaders(url string, data []byte, headers map[string]string) ([]byte, error) {
	start := time.Now()
	defer func() {
		log.Info("JsonPost", "url", url, "duration", time.Since(start))
	}()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		log.Error("JsonPost request build error", "url", url, "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := cli.Do(req)
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
