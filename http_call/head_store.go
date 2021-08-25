package http_call

import (
	"encoding/json"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/types"
	"github.com/mapprotocol/compass/utils"
	log "github.com/sirupsen/logrus"
)

func HeaderCurrentNumber(url string, chainEnum chains.ChainId) (num uint64) {
	requestBody, _ := json.Marshal(types.Request{
		Method: "header_currentHeaderNumber",
		Params: []chains.ChainId{chainEnum},
		Id:     "1",
	})
	responseByte := utils.RpcToolFromRequestByte2ResponseByte(&url, &requestBody)
	err := json.Unmarshal(*responseByte, &struct {
		Result *uint64 `json:"result"`
	}{&num})
	if err != nil {
		log.Warnln("HeaderCurrentNumber error: ", err)
		return ^uint64(0)
	}
	return
}
