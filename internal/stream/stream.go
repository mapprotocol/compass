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
	SrcChain    string `json:"src_chain"`
	SrcTxHash   string `json:"src_tx_hash"`
	SrcLogIndex uint   `json:"src_log_index"`
	BlockNumber int64  `json:"block_number"`
	DesChain    string `json:"des_chain"`
}

type TxExecOfRequest struct {
	SrcChain         string `json:"src_chain"`
	SrcTxHash        string `json:"src_tx_hash"`
	SrcLogIndex      uint   `json:"src_log_index"`
	SrcBlockNumber   int64  `json:"src_block_number"`
	RelayChain       string `json:"relay_chain"`
	RelayTxHash      string `json:"relay_tx_hash"`
	RelayLogIndex    uint   `json:"relay_log_index"`
	RelayBlockNumber int64  `json:"relay_block_number"`
	Status           int64  `json:"status"`
	DesChain         string `json:"des_chain"`
	DesTxHash        string `json:"des_tx_hash"`
	DesLogIndex      uint   `json:"des_log_index"`
}
