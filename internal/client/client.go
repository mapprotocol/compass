package client

import (
	"bytes"
	"io"
	"net/http"

	log "github.com/ChainSafe/log15"
)

func JsonPost(url string, data []byte) ([]byte, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
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
	resp, err := http.Get(url)
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
