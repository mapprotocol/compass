package chain_structs

type GetAccountInfoResponse struct {
	EpochID        string `json:"epochID"`
	RegisterStatus bool   `json:"registerStatus"`
	RelayerStatus  bool   `json:"relayerStatus"`
}
