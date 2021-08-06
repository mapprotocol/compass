package chain_structs

import "math/big"

type GetRelayerBalanceResponse struct {
	Register *big.Int
	Locked   *big.Int
	Unlocked *big.Int
}
