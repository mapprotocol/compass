package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

var (
	prefix, hooksUrl = "", ""
	monitor          = make(map[string]int64)
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
	if v, ok := monitor[msg]; ok {
		if time.Now().Unix()-v < 300 { // ignore same alarm in five minute
			return
		}
	}
	monitor[msg] = time.Now().Unix()
	body, err := json.Marshal(map[string]interface{}{
		"text": fmt.Sprintf("%s %s", prefix, msg),
	})
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "POST", hooksUrl, ioutil.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warn("read resp failed", "err", err)
		return
	}
	log.Info("send alarm message", "resp", string(data))
}
