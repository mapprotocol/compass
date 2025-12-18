package butter

import (
	"encoding/json"
	"fmt"
	"math/big"

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

func SolCrossIn(domain, query string) (*SolCrossInResp, error) {
	return defaultButter.SolCrossIn(domain, query)
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

func GetVolume(domain, txHash, _type string) (int64, error) {
	return defaultButter.TranscationVerify(domain, fmt.Sprintf("hash=%s&type=%s", txHash, _type))
}

type TransactionVerifyResponse struct {
	Errno   int    `json:"errno"`
	Message string `json:"message"`
	Data    struct {
		Volume string `json:"volume"`
	} `json:"data"`
}

func (b *Butter) TranscationVerify(domain, query string) (int64, error) {
	uri := fmt.Sprintf("%s%s?%s", domain, UrlOfTransactionVerify, query)
	fmt.Println("TransactionVerify uri ", uri)
	body, err := client.JsonGet(uri)
	if err != nil {
		return 0, err
	}

	data := TransactionVerifyResponse{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}
	if data.Errno != 0 {
		return 0, fmt.Errorf("code %d, mess:%s", data.Errno, data.Message)
	}
	if data.Data.Volume == "" {
		return 0, nil
	}

	volume, ok := big.NewInt(0).SetString(data.Data.Volume, 10)
	if !ok {
		return 0, fmt.Errorf("failed to parse volume")
	}
	volume = volume.Div(volume, big.NewInt(1000000))

	return volume.Int64(), nil
}
