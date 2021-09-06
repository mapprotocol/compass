package types

import "math/big"

type EventLogSwapOutResponse struct {
	OrderId     *big.Int `json:"orderId"`
	Amount      *big.Int `json:"amount"`
	FromChainID *big.Int `json:"fromChainID"`
	ToChainID   *big.Int `json:"toChainID"`
}
