package cmd_runtime

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/types"
	"time"
)

var (
	EventSwapOutKey         = "EventSwapOutKey"
	EventSwapOutArrayKey    = "eventSwapOutArrayKey"
	EventSwapOutHash        = crypto.Keccak256Hash([]byte("LogSwapOut(uint256,address,address,address,uint256,uint256,uint256)"))
	GlobalConfigV           types.GlobalConfig
	SrcChainConfig          types.ChainConfig
	DstChainConfig          types.ChainConfig
	DstInstance             chains.ChainInterface
	SrcInstance             chains.ChainInterface
	BlockNumberByEstimation = true

	StructRegisterNotRelayer = &types.WaitTimeAndMessage{
		Time:    2 * time.Minute,
		Message: "registered not relayer",
	}
	StructUnregistered = &types.WaitTimeAndMessage{
		Time:    10 * time.Minute,
		Message: "Unregistered",
	}
	StructUnStableBlock = &types.WaitTimeAndMessage{
		Time:    time.Second * 2, //it will update at InitConfigAndClient func
		Message: "Unstable block",
	}
)
