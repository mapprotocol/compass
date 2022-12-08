package eth2

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mapprotocol/compass/internal/bsc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

type Maintainer struct {
	*chain.BaseMaintainer
}

//func NewMaintainer(cs *chain.CommonSync) *Maintainer {
//	chain.NewBaseMaintainer(cs, cs.Cfg.BlockConfirmations, nil, syncHeaderToMap)
//}

// syncHeaderToMap listen header from current chain to Map chain
func syncHeaderToMap(m chain.BaseMaintainer, latestBlock *big.Int) error {
	// epoch check

	// synced height check
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if latestBlock.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", latestBlock)
		return nil
	}
	m.Log.Info("find sync block", "current height", latestBlock)
	headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
	for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
		headerHeight := new(big.Int).Sub(latestBlock, new(big.Int).SetInt64(int64(i)))
		header, err := m.Conn.Client().HeaderByNumber(context.Background(), headerHeight)
		if err != nil {
			return err
		}
		headers[mapprotocol.HeaderCountOfBsc-i-1] = *header
	}

	params := make([]bsc.Header, 0, len(headers))
	for _, h := range headers {
		params = append(params, bsc.ConvertHeader(h))
	}
	input, err := mapprotocol.Bsc.Methods[mapprotocol.MethodOfGetHeadersBytes].Inputs.Pack(params)
	if err != nil {
		m.Log.Error("Failed to abi pack", "err", err)
		return err
	}

	id := big.NewInt(0).SetUint64(uint64(m.Cfg.Id))
	msgpayload := []interface{}{id, input}
	message := msg.NewSyncToMap(m.Cfg.Id, m.Cfg.MapChainID, msgpayload, m.MsgCh)

	err = m.Router.Send(message)
	if err != nil {
		m.Log.Error("Subscription error: failed to route message", "err", err)
		return err
	}

	err = m.WaitUntilMsgHandled(1)
	if err != nil {
		return err
	}
	return nil
}
