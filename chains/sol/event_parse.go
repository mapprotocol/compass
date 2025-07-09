package sol

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
)

// CrossFinishEvent contains the parameters for a gateway deposit instruction
type CrossFinishEvent struct {
	Discriminator [8]byte
	CrossType     string
	AfterBalance  uint64
	AmountOut     uint64
	OrderRecord   *OrderRecord
}

type OrderRecord struct {
	OrderId                   []byte
	Payer                     solana.PublicKey
	FromChainId               uint64
	ToChainId                 uint64
	ToToken                   [32]byte
	FromToken                 [32]byte
	From                      [32]byte
	Receiver                  [32]byte
	TokenAmount               *big.Int
	SwapTokenOut              solana.PublicKey
	SwapTokenOutBeforeBalance uint64
	SwapTokenOutMinAmountOut  uint64
	MinAmountOut              *big.Int
	RefererId                 []int64
	FeeRatio                  []int64
}

func parseCrossFinishEventData(data []byte) (*CrossFinishEvent, error) {
	if len(data) < 170 {
		return nil, fmt.Errorf("event length not enouth: %d < 170", len(data))
	}
	if data[0] != 201 {
		return nil, fmt.Errorf("event not finish, %v", data[0])
	}
	offset := 9
	afterBalance := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	amountOut := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	orderId := make([]byte, 32)
	copy(orderId, data[offset:offset+32])
	offset += 32

	var payer solana.PublicKey
	copy(payer[:], data[offset:offset+32])
	offset += 32

	orderRecord := &OrderRecord{
		OrderId: orderId,
		Payer:   payer,
	}

	return &CrossFinishEvent{
		AfterBalance: afterBalance,
		AmountOut:    amountOut,
		OrderRecord:  orderRecord,
	}, nil
}
