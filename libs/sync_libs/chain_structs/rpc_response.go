package chain_structs

import "math/big"

type GetRelayerBalanceResponse struct {
	Register *big.Int
	Locked   *big.Int
	Unlocked *big.Int
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
