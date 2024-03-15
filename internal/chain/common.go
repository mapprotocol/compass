package chain

import (
	"fmt"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/msg"
	"github.com/pkg/errors"
	"math/big"
)

var (
	OrderExist    = errors.New("order exist")
	NotVerifyAble = errors.New("not verify able")
)

type OrderStatusResp struct {
	Exists     bool
	Verifiable bool
	NodeType   *big.Int
}

func OrderStatus(idx int, selfChainId, toChainID uint64, blockNumber *big.Int, orderId []byte) (*OrderStatusResp, error) {
	call, ok := mapprotocol.ContractMapping[msg.ChainId(toChainID)]
	if !ok {

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
	fmt.Println("ret ", ret)
	if ret.Exists {
		return 0, OrderExist
	}
	if !ret.Verifiable {
		return 0, NotVerifyAble
	}

	return ret.NodeType.Int64(), nil
}
