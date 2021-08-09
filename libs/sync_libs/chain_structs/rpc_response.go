package chain_structs

import "math/big"

type GetRelayerBalanceResponse struct {
	Registered    *big.Int
	Unregistering *big.Int
	Unregistered  *big.Int
}
type GetRelayerResponse struct {
	Relayer  bool
	Register bool
	Epoch    *big.Int
}
type GetPeriodHeightResponse struct {
	Start   *big.Int
	End     *big.Int
	Relayer bool
}
