package butter

import (
	"encoding/json"
	"fmt"

	"github.com/mapprotocol/compass/internal/client"
)

const (
	UrlOfExecSwap       = "/execSwap"
	UrlOfSolCrossIn     = "/solanaCrossIn"
	UrlOfRetryMessageIn = "/retryMessageIn"
)

var defaultButter = New()

type Butter struct {
}

func New() *Butter {
	return &Butter{}
}

func (b *Butter) ExecSwap(domain, query string) ([]byte, error) {
	return client.JsonGet(fmt.Sprintf("%s%s?%s", domain, UrlOfExecSwap, query))
}

func (b *Butter) RetryMessageIn(domain, query string) ([]byte, error) {
	return client.JsonGet(fmt.Sprintf("%s%s?%s", domain, UrlOfRetryMessageIn, query))
}

func (b *Butter) SolCrossIn(domain, query string) (*SolCrossInResp, error) {
	fmt.Println("SolCrossIn uri ", fmt.Sprintf("%s%s?%s", domain, UrlOfSolCrossIn, query))
	body, err := client.JsonGet(fmt.Sprintf("%s%s?%s", domain, UrlOfSolCrossIn, query))
	if err != nil {
		return nil, err
	}
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

	if data.Data[0].Error.Response.Errno != 0 {
		return nil, fmt.Errorf("code %d, mess:%s", data.Data[0].Error.Response.Errno, data.Data[0].Error.Response.Message)
	}

	if len(data.Data[0].TxParam) <= 0 {
		return nil, fmt.Errorf("data txParam is zero")
	}

	return &data, nil
}

type BlockedAccountResponse struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    bool   `json:"data"` // 使用 interface{} 可以接收任何类型
}

func (b *Butter) BlockedAccount(domain, query string) (bool, error) {
	uri := fmt.Sprintf("%s%s?%s", domain, UrlOfSolCrossIn, query)
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

func SolCrossIn(domain, query string) (*SolCrossInResp, error) {
	return defaultButter.SolCrossIn(domain, query)
}

func BlockedAccount(domain, sourceChain, initiator, from string) (bool, error) {
	if initiator != "" {
		query := fmt.Sprintf("account=%s&chainId=%s", initiator, sourceChain)
		isBlock, err := defaultButter.BlockedAccount(domain, query)
		if err != nil {
			return false, err
		}
		if isBlock {
			return true, nil
		}
	}
	if from != "" {
		query := fmt.Sprintf("account=%s&chainId=%s", from, sourceChain)
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
