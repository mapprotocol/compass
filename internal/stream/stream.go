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

type FailedTxOfRequest struct {
	ToChain string `json:"to_chain"`
	Hash    string `json:"hash"`
}

type ProofOfRequest struct {
	SrcChain    int    `json:"src_chain"`
	SrcTxHash   string `json:"src_tx_hash"`
	SrcLogIndex int    `json:"src_log_index"`
	DesChain    int    `json:"des_chain"`
}
