package chain

import (
	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/mapprotocol/compass/blockstore"
	"github.com/mapprotocol/compass/mapprotocol"
)

// SetupBlockStore queries the blockstore for the latest known block. If the latest block is
// greater than cfg.startBlock, then cfg.startBlock is replaced with the latest known block.
func SetupBlockStore(cfg *Config, kp *secp256k1.Keypair, role mapprotocol.Role) (*blockstore.Blockstore, error) {
	bs, err := blockstore.NewBlockstore(cfg.BlockstorePath, cfg.Id, kp.Address(), role)
	if err != nil {
		return nil, err
	}

	if !cfg.FreshStart {
		latestBlock, err := bs.TryLoadLatestBlock()
		if err != nil {
			return nil, err
		}

		if latestBlock.Cmp(cfg.StartBlock) == 1 {
			cfg.StartBlock = latestBlock
		}
	}

	return bs, nil
}
