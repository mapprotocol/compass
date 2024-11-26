package sol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
)

type Pubkey [32]byte

type RouteOrder struct {
	OrderID     [32]byte
	Payer       Pubkey
	User        Pubkey
	FromChainID uint64
	ToChainID   uint64
	FromToken   Pubkey
	TokenAmount uint128 // uint128 is implemented as a custom struct
	AmountOut   uint64
	SwapData    []byte
}

func (ro *RouteOrder) Encode() ([]byte, error) {
	buffer := new(bytes.Buffer)

	// Write the fields into the buffer
	buffer.Write(ro.OrderID[:])
	buffer.Write(ro.Payer[:])
	buffer.Write(ro.User[:])
	binary.Write(buffer, binary.LittleEndian, ro.FromChainID)
	binary.Write(buffer, binary.LittleEndian, ro.ToChainID)
	buffer.Write(ro.FromToken[:])
	binary.Write(buffer, binary.LittleEndian, ro.TokenAmount)
	binary.Write(buffer, binary.LittleEndian, ro.AmountOut)
	buffer.Write(ro.SwapData)

	return buffer.Bytes(), nil
}

func DecodeRouteOrder(data []byte) (*RouteOrder, error) {
	if len(data) < 138 { // Minimum size based on structure fields
		return nil, errors.New("invalid message length")
	}

	offset := 0

	// Decode OrderID
	var orderID [32]byte
	copy(orderID[:], data[offset:offset+32])
	offset += 32

	// Decode Payer
	var payer Pubkey
	copy(payer[:], data[offset:offset+32])
	offset += 32

	// Decode User
	var user Pubkey
	copy(user[:], data[offset:offset+32])
	offset += 32

	// Decode FromChainID
	fromChainID := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Decode ToChainID
	toChainID := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Decode FromToken
	var fromToken Pubkey
	copy(fromToken[:], data[offset:offset+32])
	offset += 32

	// Decode TokenAmount
	tokenAmount := uint128{
		Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
		Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
	}
	offset += 16

	// Decode AmountOut
	amountOut := binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Decode SwapData
	swapData := data[offset:]

	return &RouteOrder{
		OrderID:     orderID,
		Payer:       payer,
		User:        user,
		FromChainID: fromChainID,
		ToChainID:   toChainID,
		FromToken:   fromToken,
		TokenAmount: tokenAmount,
		AmountOut:   amountOut,
		SwapData:    swapData,
	}, nil
}

// uint128 is a placeholder for a 128-bit unsigned integer.
type uint128 struct {
	Lo uint64
	Hi uint64
}

func (u *uint128) Bytes() []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:8], u.Lo)
	binary.LittleEndian.PutUint64(buf[8:16], u.Hi)
	return buf
}

func (u *uint128) String() string {
	return fmt.Sprintf("Hi: %d, Lo: %d", u.Hi, u.Lo)
}

type SwapData struct {
	ToToken      []byte   `json:"toToken"`
	Receiver     []byte   `json:"receiver"`
	Initiator    []byte   `json:"initiator"`
	MinAmountOut *big.Int `json:"minAmountOut"`
	Relay        bool     `json:"relay"`
	MessageType  *big.Int `json:"messageType"`
}

func parseSwapData(data []byte) (*SwapData, error) {
	offset := 0
	if len(data) < 74 { // 最小长度验证
		return nil, errors.New("data too short to parse")
	}

	// 解析 cross_token_len
	crossTokenLen := int(data[offset])
	offset++

	// 解析 cross_address_len
	crossAddressLen := int(data[offset])
	offset++

	// 验证剩余长度是否足够
	expectedLength := 2 + crossTokenLen + 2*crossAddressLen + 32 + 2
	if len(data) < expectedLength {
		return nil, errors.New("data length mismatch")
	}

	// 解析 toToken
	toToken := data[offset : offset+crossTokenLen]
	offset += crossTokenLen

	// 解析 receiver
	receiver := data[offset : offset+crossAddressLen]
	offset += crossAddressLen

	// 解析 initiator
	initiator := data[offset : offset+crossAddressLen]
	offset += crossAddressLen

	// 解析 minAmountOut (128 位整数)
	minAmountOut := binary.BigEndian.Uint64(data[offset+24 : offset+32]) // 低 8 字节
	offset += 32

	// 解析 relay
	relay := data[offset]
	offset++

	// 解析 messageType
	messageType := data[offset]

	// 返回解析结果
	relayBool, err := strconv.ParseBool(fmt.Sprintf("%x", relay))
	if err != nil {
		return nil, err
	}

	return &SwapData{
		ToToken:      toToken,
		Receiver:     receiver,
		Initiator:    initiator,
		MinAmountOut: big.NewInt(0).SetUint64(minAmountOut),
		Relay:        relayBool,
		MessageType:  big.NewInt(0).SetBytes([]byte{messageType}),
	}, nil
}
