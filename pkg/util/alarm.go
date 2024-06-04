package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

var (
	prefix, hooksUrl = "", ""
	m                = NewRWMap()
)

func Init(env, hooks string) {
	prefix = env
	hooksUrl = hooks
}

func Alarm(ctx context.Context, msg string) {
	if hooksUrl == "" {
		log.Info("hooks is empty")
		return
	}
	fmt.Println("send alarm in")
	v, ok := m.Get(msg)
	if ok {
		if time.Now().Unix()-v < 300 { // ignore same alarm in five minute
			return
		}
	}

	m.Set(msg, time.Now().Unix())
	body, err := json.Marshal(map[string]interface{}{
		"text": fmt.Sprintf("%s %s", prefix, msg),
	})
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hooksUrl, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn("read resp failed", "err", err)
		return
	}
	fmt.Println("send alarm message", "resp", string(data))
}
