package sync_libs

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"signmap/libs/sync_libs/chain_structs"
)

func rpcToolFromRequestByte2ResponseByte(url *string, requestByte *[]byte) *[]byte {
	requestBodyBytes := bytes.NewBuffer(*requestByte)
	resp, err := http.Post(*url, "application/json", requestBodyBytes)
	nilResp := []byte{}
	if err != nil {
		return &nilResp
	}
	body, err := ioutil.ReadAll(resp.Body)
	err = resp.Body.Close()
	if err != nil {
		return nil
	}
	return &body
}
func HeaderCurrentNumber(url string, chianEnum chain_structs.ChainEnum) (num int) {
	requestBody, _ := json.Marshal(chain_structs.Request{
		Method: "header_currentHeaderNumber",
		Params: []chain_structs.ChainEnum{chianEnum},
		Id:     "1",
	})
	responseByte := rpcToolFromRequestByte2ResponseByte(&url, &requestBody)
	err := json.Unmarshal(*responseByte, &struct {
		Result *int `json:"result"`
	}{&num})
	if err != nil {
		return 0
	}
	return
}
