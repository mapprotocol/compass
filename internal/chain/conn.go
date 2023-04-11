package chain

import (
	"math/big"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/eth2"
	"github.com/mapprotocol/compass/internal/klaytn"

	"github.com/ChainSafe/chainbridge-utils/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/pkg/ethclient"
)

type Connection interface {
	Connect() error
	Keypair() *secp256k1.Keypair
	Opts() *bind.TransactOpts
	CallOpts() *bind.CallOpts
	LockAndUpdateOpts(bool) error
	UnlockOpts()
	Client() *ethclient.Client
	EnsureHasBytecode(address common.Address) error
	LatestBlock() (*big.Int, error)
	WaitForBlock(block *big.Int, delay *big.Int) error
	Close()
}

type KConnection interface {
	Connection
	KClient() *klaytn.Client
}

type Eth2Connection interface {
	Connection
	Eth2Client() *eth2.Client
}

type CreateConn func(string, bool, *secp256k1.Keypair, log15.Logger, *big.Int, *big.Int, float64, string, string) Connection
