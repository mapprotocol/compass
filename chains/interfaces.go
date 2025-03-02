package chains

import (
	"github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/chains/bsc"
	"github.com/mapprotocol/compass/chains/btc"
	"github.com/mapprotocol/compass/chains/conflux"
	"github.com/mapprotocol/compass/chains/eth2"
	"github.com/mapprotocol/compass/chains/ethereum"
	"github.com/mapprotocol/compass/chains/klaytn"
	"github.com/mapprotocol/compass/chains/matic"
	"github.com/mapprotocol/compass/chains/near"
	"github.com/mapprotocol/compass/chains/sol"
	"github.com/mapprotocol/compass/chains/ton"
	"github.com/mapprotocol/compass/chains/tron"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/constant"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

var (
	chainMap = map[string]Chainer{
		constant.Bsc:      bsc.New(),
		constant.Matic:    matic.New(),
		constant.Conflux:  conflux.New(),
		constant.Eth2:     eth2.New(),
		constant.Ethereum: ethereum.New(),
		constant.Klaytn:   klaytn.New(),
		constant.Near:     near.New(),
		constant.Solana:   sol.New(),
		constant.Ton:      ton.New(),
		constant.Tron:     tron.New(),
		constant.Btc:      btc.New(),
	}
	proofMap = map[string]Proffer{
		constant.Bsc:      bsc.New(),
		constant.Matic:    matic.New(),
		constant.Conflux:  conflux.New(),
		constant.Eth2:     eth2.New(),
		constant.Ethereum: ethereum.New(),
		constant.Klaytn:   klaytn.New(),
		constant.Tron:     tron.New(),
	}
)

func Create(_type string) (Chainer, bool) {
	if chain, ok := chainMap[_type]; ok {
		return chain, true
	}
	return nil, false
}

type Chainer interface {
	New(*core.ChainConfig, log15.Logger, chan<- error, mapprotocol.Role) (core.Chain, error)
}

func CreateProffer(_type string) (Proffer, bool) {
	if chain, ok := proofMap[_type]; ok {
		return chain, true
	}
	return nil, false
}

type Proffer interface {
	Connect(id, endpoint, mcs, lightNode, oracleNode string) (*ethclient.Client, error)
	Proof(client *ethclient.Client, log *types.Log, endpoint string, proofType int64, selfId,
		toChainID uint64, sign [][]byte) ([]byte, error)
	Maintainer(client *ethclient.Client, selfId, toChainId uint64, srcEndpoint string) ([]byte, error)
}
