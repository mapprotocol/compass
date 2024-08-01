package tron

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func GetOrderPrice(key string, resourceValue, rentDuration int64) (*GetOrderPriceResp, error) {
	url := fmt.Sprintf("https://feee.io/open/v2/order/price?resource_value=%d&rent_duration=%d&rent_time_unit=d", resourceValue, rentDuration)
	fmt.Println("GetOrderPrice url is", url)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, err
	}
	req.Header.Add("key", key)
	req.Header.Add("User-Agent", "Feee.io Client/1.0.0 (https://feee.io)")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println("GetOrderPrice back", string(body))
	ret := GetOrderPriceResp{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func OrderSubmit(key, addr string, resValue, rentDuration int64) (*OrderResp, error) {
	url := "https://feee.io/open/v2/order/submit"
	method := "POST"

	data := map[string]interface{}{
		"resource_type":   1,
		"receive_address": addr,
		"resource_value":  resValue,
		"rent_duration":   rentDuration,
		"rent_time_unit":  "d",
	}
	by, _ := json.Marshal(data)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewReader(by))

	if err != nil {
		return nil, err
	}
	req.Header.Add("key", key)
	req.Header.Add("User-Agent", "Feee.io Client/1.0.0 (https://feee.io)")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println("OrderSubmit back", string(body))
	ret := OrderResp{}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type GetOrderPriceResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Data      struct {
		ResourceValue int     `json:"resource_value"`
		PayAmount     float64 `json:"pay_amount"`
		RentDuration  int     `json:"rent_duration"`
		RentTimeUnit  string  `json:"rent_time_unit"`
		PriceInSun    int     `json:"price_in_sun"`
	} `json:"data"`
}

type OrderResp struct {
	Code      int    `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Data      struct {
		OrderNo             string  `json:"order_no"`
		OrderType           int     `json:"order_type"`
		ResourceType        int     `json:"resource_type"`
		ReceiveAddress      string  `json:"receive_address"`
		PriceInSun          int     `json:"price_in_sun"`
		MinAmount           int     `json:"min_amount"`
		MinPayout           int     `json:"min_payout"`
		MinFreeze           int     `json:"min_freeze"`
		MaxAmount           int     `json:"max_amount"`
		MaxPayout           float64 `json:"max_payout"`
		MaxFreeze           int     `json:"max_freeze"`
		FreezeTime          int     `json:"freeze_time"`
		UnfreezeTime        int     `json:"unfreeze_time"`
		ExpireTime          int     `json:"expire_time"`
		CreateTime          int     `json:"create_time"`
		ResourceValue       int     `json:"resource_value"`
		ResourceSplitValue  int     `json:"resource_split_value"`
		FrozenResourceValue int     `json:"frozen_resource_value"`
		RentDuration        int     `json:"rent_duration"`
		RentTimeUnit        string  `json:"rent_time_unit"`
		RentExpireTime      int     `json:"rent_expire_time"`
		FrozenBalance       int     `json:"frozen_balance"`
		FrozenTxID          string  `json:"frozen_tx_id"`
		UnfreezeTxID        string  `json:"unfreeze_tx_id"`
		SettleAmount        float64 `json:"settle_amount"`
		SettleAddress       string  `json:"settle_address"`
		SettleTime          int     `json:"settle_time"`
		PayTime             int     `json:"pay_time"`
		PayAmount           float64 `json:"pay_amount"`
		RefundAmount        int     `json:"refund_amount"`
		RefundTime          int     `json:"refund_time"`
		IsSplit             int     `json:"is_split"`
		Status              int     `json:"status"`
	} `json:"data"`
}
