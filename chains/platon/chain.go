package platon

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"

	metrics "github.com/ChainSafe/chainbridge-utils/metrics/types"
	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/core"
	"github.com/mapprotocol/compass/internal/bsc"
	"github.com/mapprotocol/compass/internal/chain"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
)

func InitializeChain(chainCfg *core.ChainConfig, logger log15.Logger, sysErr chan<- error, m *metrics.ChainMetrics,
	role mapprotocol.Role) (core.Chain, error) {
	return chain.New(chainCfg, logger, sysErr, m, role, chain.OptOfSync2Map(syncHeaderToMap))
}

func syncHeaderToMap(m *chain.Maintainer, needSyncHeight *big.Int) error {
	remainder := big.NewInt(0).Mod(new(big.Int).Sub(needSyncHeight, new(big.Int).SetInt64(mapprotocol.HeaderCountOfBsc-1)),
		big.NewInt(mapprotocol.EpochOfBsc))
	if remainder.Cmp(mapprotocol.Big0) != 0 {
		return nil
	}
	// synced height check
	syncedHeight, err := mapprotocol.Get2MapHeight(m.Cfg.Id)
	if err != nil {
		m.Log.Error("Get current synced Height failed", "err", err)
		return err
	}
	if needSyncHeight.Cmp(syncedHeight) <= 0 {
		m.Log.Info("CurrentBlock less than synchronized headerHeight", "synced height", syncedHeight,
			"current height", needSyncHeight)
		return nil
	}
	m.Log.Info("find sync block", "current height", needSyncHeight)
	headers := make([]types.Header, mapprotocol.HeaderCountOfBsc)
	for i := 0; i < mapprotocol.HeaderCountOfBsc; i++ {
		headerHeight := new(big.Int).Sub(needSyncHeight, new(big.Int).SetInt64(int64(i)))
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
