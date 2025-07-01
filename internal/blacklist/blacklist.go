package blacklist

import (
	"fmt"
	"github.com/mapprotocol/compass/internal/client"
	"github.com/pkg/errors"
	"strconv"
)

type Blacklist interface {
	CheckAccount(account string) (bool, error)
	CheckTxs(chainId, txHash string) (bool, error)
}

type blockList struct {
	domain string
}

var defaultBlockList *blockList

const (
	UrlOfCheckAccount = "/blocklist/blockedAccount"
	UrlOfCheckTx      = "/blocklist/blockedTxn"
)

func Init(domain string) {
	defaultBlockList = &blockList{domain: domain}
}

func (b *blockList) CheckAccount(account, chainId string) (bool, error) {
	uri := fmt.Sprintf("%s%s?account=%s&chainId=%s", b.domain, UrlOfCheckAccount, account, chainId)
	body, err := client.JsonGet(uri)
	if err != nil {
		return false, errors.Wrap(err, "CheckAccount JsonGet")
	}

	ret, err := strconv.ParseBool(string(body))
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("CheckAccount, account: %s body: %s", account, string(body)))
	}
	return ret, nil
}

func (b *blockList) CheckTxs(chainId, txHash string) (bool, error) {
	uri := fmt.Sprintf("%s%s?chainId=%s&txHash=%s", b.domain, UrlOfCheckTx, chainId, txHash)
	body, err := client.JsonGet(uri)
	if err != nil {
		return false, errors.Wrap(err, "CheckTxs JsonGet")
	}

	ret, err := strconv.ParseBool(string(body))
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("CheckTxs, chainId: %s txHash: %s body: %s", chainId, txHash, string(body)))
	}
	return ret, nil
}

func CheckTxs(chainId, txHash string) (bool, error) {
	return defaultBlockList.CheckTxs(chainId, txHash)
}

func CheckAccount(account, chainId string) (bool, error) {
	return defaultBlockList.CheckAccount(account, chainId)
}
