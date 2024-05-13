package chain

import (
	"math/big"

	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/blockstore"
)

func SetupBlockStore(cfg *Config, role mapprotocol.Role) (*blockstore.Blockstore, error) {
	if cfg.Filter {
		role += "-filter"
	}
	bs, err := blockstore.NewBlockstore(cfg.BlockstorePath, cfg.Id, cfg.From, role)
	if err != nil {
		return nil, err
	}

	if !cfg.FreshStart {
		latestBlock, err := bs.TryLoadLatestBlock()
		if err != nil {
			return nil, err
		}
		if latestBlock == nil {
			latestBlock = new(big.Int)
		}

		if latestBlock.Cmp(cfg.StartBlock) == 1 {
			cfg.StartBlock = latestBlock
		}
	}

	return bs, nil
}
