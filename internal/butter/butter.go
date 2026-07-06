package butter

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mapprotocol/compass/internal/client"
)

const (
	UrlOfExecSwap          = "/execSwap"
	UrlOfSolCrossIn        = "/solanaCrossIn"
	UrlOfRetryMessageIn    = "/retryMessageIn"
	UrlOfBlockedAccount    = "/blocklist/blockedAccount"
	UrlOfTransactionVerify = "/api/transaction/verify"
)

var defaultButter = New()
var (
	apiKeyMu sync.RWMutex
	apiKey   string
)

type Butter struct {
}

func New() *Butter {
	return &Butter{}
}

func Init(key string) {
	SetAPIKey(key)
}

func SetAPIKey(key string) {
	apiKeyMu.Lock()
	defer apiKeyMu.Unlock()
	apiKey = strings.TrimSpace(key)
}

func backendHeaders() map[string]string {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	if apiKey == "" {
		return nil
	}
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
	}
}

func jsonGetBackend(url string) ([]byte, error) {
	return client.JsonGetWithHeaders(url, backendHeaders())
}

func jsonPostBackend(url string, data []byte) ([]byte, error) {
	return client.JsonPostWithHeaders(url, data, backendHeaders())
}

func (b *Butter) ExecSwap(domain, query string) ([]byte, error) {
	return jsonGetBackend(fmt.Sprintf("%s%s?%s", domain, UrlOfExecSwap, query))
}

func (b *Butter) RetryMessageIn(domain, query string) ([]byte, error) {
	return jsonGetBackend(fmt.Sprintf("%s%s?%s", domain, UrlOfRetryMessageIn, query))
}

func (b *Butter) SolCrossIn(domain, query string, reqbody map[string]interface{}) (*SolCrossInResp, error) {
	uri := fmt.Sprintf("%s%s?%s", domain, UrlOfSolCrossIn, query)
	jsonBody, err := json.Marshal(reqbody)
	if err != nil {
		return nil, err
	}
	fmt.Println("reqbody ------------- ", string(jsonBody))
	body, err := jsonPostBackend(uri, jsonBody)
	if err != nil {
		return nil, err
	}

	fmt.Println("body ------------- ", string(body))
	data := SolCrossInResp{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Errno != 0 {
		return nil, fmt.Errorf("code %d, mess:%s", data.Errno, data.Message)
	}
	if data.StatusCode != 0 {
		return nil, fmt.Errorf("code %d, mess:%s", data.StatusCode, data.Message)
	}

	if len(data.Data) <= 0 {
		return nil, fmt.Errorf("data is zero")
	}

	for _, item := range data.Data {
		if len(item.TxParam) > 0 {
			return &data, nil
		}
	}

	for _, item := range data.Data {
		if item.Error.Response.Errno != 0 {
			return nil, fmt.Errorf("code %d, mess:%s", item.Error.Response.Errno, item.Error.Response.Message)
		}
		if item.Error.Message != "" {
			return nil, fmt.Errorf("mess:%s", item.Error.Message)
		}
	}

	return nil, fmt.Errorf("data txParam is zero")
}

type BlockedAccountResponse struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    bool   `json:"data"` // 使用 interface{} 可以接收任何类型
}

func (b *Butter) BlockedAccount(domain, query string) (bool, error) {
	uri := fmt.Sprintf("%s%s?%s", domain, UrlOfBlockedAccount, query)
	fmt.Println("BlockedAccount uri ", uri)
	body, err := client.JsonGet(uri)
	if err != nil {
		return false, err
	}

	data := BlockedAccountResponse{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, err
	}
	if data.Errno != 0 {
		return false, fmt.Errorf("code %d, mess:%s", data.Errno, data.Message)
	}

	return data.Data, nil
}

func ExecSwap(domain, query string) ([]byte, error) {
	return defaultButter.ExecSwap(domain, query)
}

func RetryMessageIn(domain, query string) ([]byte, error) {
	return defaultButter.RetryMessageIn(domain, query)
}

func SolCrossIn(domain, query string, body map[string]interface{}) (*SolCrossInResp, error) {
	return defaultButter.SolCrossIn(domain, query, body)
}

func BlockedAccount(domain, sourceChain, initiator, from, alchemypayKytCheck string) (bool, error) {
	if len(initiator) > 2 && initiator != "0x0000000000000000000000000000000000000000" {
		query := fmt.Sprintf("account=%s&chainId=%s&alchemypayKytCheck=%s", initiator, sourceChain, alchemypayKytCheck)
		isBlock, err := defaultButter.BlockedAccount(domain, query)
		if err != nil {
			return false, err
		}
		if isBlock {
			return true, nil
		}
	}
	if len(from) > 2 && from != "0x0000000000000000000000000000000000000000" {
		query := fmt.Sprintf("account=%s&chainId=%s&alchemypayKytCheck=%s", from, sourceChain, alchemypayKytCheck)
		isBlock, err := defaultButter.BlockedAccount(domain, query)
		if err != nil {
			return false, err
		}
		if isBlock {
			return true, nil
		}
	}
	return false, nil
}

func TranscationVerify(domain, txHash, _type string) (*TransactionVerifyResponse, error) {
	return defaultButter.TranscationVerify(domain, fmt.Sprintf("hash=%s&type=%s", txHash, _type))
}

type TransactionVerifyResponse struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    struct {
		Volume string `json:"volume"`
		Sender string `json:"sender"`
	} `json:"data"`
}

func (b *Butter) TranscationVerify(domain, query string) (*TransactionVerifyResponse, error) {
	uri := fmt.Sprintf("%s%s?%s", domain, UrlOfTransactionVerify, query)
	fmt.Println("TransactionVerify uri ", uri)
	body, err := client.JsonGet(uri)
	if err != nil {
		return nil, err
	}

	data := TransactionVerifyResponse{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	if data.Errno != 0 {
		return nil, fmt.Errorf("query %s, code %d, mess:%s", query, data.Errno, data.Message)
	}
	if data.Data.Volume == "" {
		return nil, nil
	}

	return &data, nil
}
