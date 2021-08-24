package utils

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func RpcToolFromRequestByte2ResponseByte(url *string, requestByte *[]byte) *[]byte {
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
