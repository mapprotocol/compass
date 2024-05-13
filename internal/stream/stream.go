package stream

type CommonResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type MosListResp struct {
	Total int64         `json:"total"`
	List  []*GetMosResp `json:"list"`
}

type GetMosResp struct {
	Id              int64  `json:"id"`
	ProjectId       int64  `json:"project_id"`
	ChainId         int64  `json:"chain_id"`
	EventId         int64  `json:"event_id"`
	TxHash          string `json:"tx_hash"`
	ContractAddress string `json:"contract_address"`
	Topic           string `json:"topic"`
	BlockNumber     uint64 `json:"block_number"`
	BlockHash       string `json:"block_hash"`
	LogIndex        uint   `json:"log_index"`
	LogData         string `json:"log_data"`
	TxIndex         uint   `json:"tx_index"`
	TxTimestamp     uint64 `json:"tx_timestamp"`
}
