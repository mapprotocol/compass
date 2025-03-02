package mapprotocol

import "math/big"

type MessageOutEvent struct {
	Relay       bool
	MessageType uint8
	FromChain   *big.Int
	ToChain     *big.Int
	OrderId     [32]byte
	Mos         []byte
	Token       []byte
	Initiator   []byte
	From        []byte
	To          []byte
	Amount      *big.Int
	GasLimit    *big.Int
	SwapData    []byte
}
