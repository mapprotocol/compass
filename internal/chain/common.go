package chain

import (
	"fmt"
	"github.com/mapprotocol/compass/internal/mapprotocol"
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/pkg/errors"
)

var (
	OrderExist       = errors.New("order exist")
	NotVerifyAble    = errors.New("not verify able")
	ContractNotExist = errors.New("contract not exist")
)

type OrderStatusResp struct {
	Exists     bool
	Verifiable bool
	NodeType   *big.Int
}

func OrderStatus(idx int, selfChainId, toChainID uint64, blockNumber *big.Int, orderId []byte) (*OrderStatusResp, error) {
	call, ok := mapprotocol.ContractMapping[msg.ChainId(toChainID)]
	if !ok {
		return nil, ContractNotExist
	}
	var fixedOrderId [32]byte
	for i, v := range orderId {
		fixedOrderId[i] = v
	}
	ret := OrderStatusResp{}
	err := call.Call(mapprotocol.MethodOfOrderStatus, &ret, idx, big.NewInt(int64(selfChainId)), blockNumber, fixedOrderId)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func PreSendTx(idx int, selfChainId, toChainID uint64, blockNumber *big.Int, orderId []byte) (int64, error) {
	ret, err := OrderStatus(idx, selfChainId, toChainID, blockNumber, orderId)
	if err != nil {
		return 0, errors.Wrap(err, "OrderStatus failed")
	}

	if ret.Exists {
		return ret.NodeType.Int64(), OrderExist
	}
	if !ret.Verifiable {
		return ret.NodeType.Int64(), NotVerifyAble
	}

	return ret.NodeType.Int64(), nil
}

type MulSignInfoResp struct {
	Version [32]byte
	Quorum  *big.Int
	Singers []common.Address
}

func MulSignInfo(idx int, toChainID uint64) (*MulSignInfoResp, error) {
	call, ok := mapprotocol.SingMapping[msg.ChainId(toChainID)]
	if !ok {
		return nil, ContractNotExist
	}

	ret := MulSignInfoResp{}
	err := call.Call(mapprotocol.MethodOfMulSignInfo, &ret, idx)
	if err != nil {
		return nil, fmt.Errorf("call failed:err:%v", err)
	}
	return &ret, nil
}

type ProposalInfoResp struct {
	Singers    []common.Address
	Signatures [][]byte
	CanVerify  bool
}

func ProposalInfo(idx int, selfChainId, toChainID uint64, blockNumber *big.Int, receipt common.Hash, version [32]byte) (*ProposalInfoResp, error) {
	call, ok := mapprotocol.SingMapping[msg.ChainId(toChainID)]
	if !ok {
		return nil, ContractNotExist
	}
	ret := ProposalInfoResp{}
	fmt.Println("MethodOfProposalInfo request ", big.NewInt(int64(selfChainId)), blockNumber, receipt, version)
	err := call.Call(mapprotocol.MethodOfProposalInfo, &ret, idx, big.NewInt(int64(selfChainId)), blockNumber, receipt, version)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
