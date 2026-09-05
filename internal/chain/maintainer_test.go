package chain

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/log15"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/internal/observability"
	"github.com/mapprotocol/compass/pkg/msg"
)

func TestMaintainerInitializesObservedCurrentBlockFromMap2OtherHeight(t *testing.T) {
	previousHeights := mapprotocol.SyncOtherMap
	mapprotocol.SyncOtherMap = map[msg.ChainId]*big.Int{
		1: big.NewInt(25_999_999),
	}
	defer func() { mapprotocol.SyncOtherMap = previousHeights }()

	stop := make(chan int)
	close(stop)
	state := observability.New("test", observability.Config{}).RegisterChain("map", "maintainer")
	maintainer := NewMaintainer(&CommonSync{
		Cfg: Config{
			Id:                 22776,
			MapChainID:         22776,
			StartBlock:         big.NewInt(0),
			BlockConfirmations: big.NewInt(0),
		},
		Log:    log15.New(),
		Stop:   stop,
		State:  state,
		height: 1,
	})

	_ = maintainer.sync()

	if state.CurrentBlock != 26_000_000 {
		t.Fatalf("observed current block = %d, want 26000000", state.CurrentBlock)
	}
}
