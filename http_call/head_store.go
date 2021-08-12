package http_call

import (
	"encoding/json"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/types"
	"github.com/mapprotocol/compass/utils"
)

func HeaderCurrentNumber(url string, chainEnum chains.ChainEnum) (num uint64) {
	requestBody, _ := json.Marshal(types.Request{
		Method: "header_currentHeaderNumber",
		Params: []chains.ChainEnum{chainEnum},
		Id:     "1",
	})
	responseByte := utils.RpcToolFromRequestByte2ResponseByte(&url, &requestBody)
	err := json.Unmarshal(*responseByte, &struct {
		Result *uint64 `json:"result"`
	}{&num})
	if err != nil {
		return 0
	}
	return
}
