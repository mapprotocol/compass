package sync_libs

import (
	"bytes"
	"encoding/json"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/types"
	"io/ioutil"
	"net/http"
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
func HeaderCurrentNumber(url string, chainEnum chains.ChainEnum) (num uint64) {
	requestBody, _ := json.Marshal(types.Request{
		Method: "header_currentHeaderNumber",
		Params: []chains.ChainEnum{chainEnum},
		Id:     "1",
	})
	responseByte := rpcToolFromRequestByte2ResponseByte(&url, &requestBody)
	err := json.Unmarshal(*responseByte, &struct {
		Result *uint64 `json:"result"`
	}{&num})
	if err != nil {
		return 0
	}
	return
}
