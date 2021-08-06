package chain_structs

type Request struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	JsonRpc string      `json:"jsonrpc"`
	Id      string      `json:"id"`
}
