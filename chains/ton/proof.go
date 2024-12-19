package ton

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"log"
	"math/big"
)

var (
	bytesType, _      = abi.NewType("bytes", "string", nil)
	messageOutType, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{
			Name: "relay",
			Type: "bool",
		},
		{
			Name: "messageType",
			Type: "uint8",
		},
		{
			Name: "fromChain",
			Type: "uint256",
		},
		{
			Name: "toChain",
			Type: "uint256",
		},
		{
			Name: "orderId",
			Type: "bytes32",
		},
		{
			Name: "mos",
			Type: "bytes",
		},
		{
			Name: "token",
			Type: "bytes",
		},
		{
			Name: "initiator",
			Type: "bytes",
		},
		{
			Name: "from",
			Type: "bytes",
		},
		{
			Name: "to",
			Type: "bytes",
		},
		{
			Name: "amount",
			Type: "uint256",
		},
		{
			Name: "gasLimit",
			Type: "uint256",
		},
		{
			Name: "swapData",
			Type: "bytes",
		},
	})
)

var (
	proofArgs = abi.Arguments{
		{Type: bytesType},
		{Type: bytesType},
		{Type: bytesType},
	}

	messageOutArgs = abi.Arguments{
		{
			Name: "messageOut",
			Type: messageOutType,
		},
	}
)

type Log struct {
	Id          int64  `json:"id"`
	BlockNumber int64  `json:"blockNumber"`
	Addr        string `json:"addr"`
	Topic       string `json:"topic"`
	Data        string `json:"data"`
	TxHash      string `json:"txHash"`
}

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

func parseMessageOutEvent(slice *cell.Slice) (*MessageOutEvent, error) {
	data1, err := slice.LoadRef()
	if err != nil {
		return nil, err
	}
	relay, err := data1.LoadBigUInt(8)
	if err != nil {
		return nil, err
	}
	msgType, err := data1.LoadBigUInt(8)
	if err != nil {
		return nil, err
	}
	fromChain, err := data1.LoadBigUInt(64)
	if err != nil {
		return nil, err
	}
	toChain, err := data1.LoadBigUInt(64)
	if err != nil {
		return nil, err
	}
	gasLimit, err := data1.LoadBigUInt(64)
	if err != nil {
		return nil, err
	}
	initiator, err := data1.LoadAddr()
	if err != nil {
		return nil, err
	}
	sender, err := data1.LoadAddr()
	if err != nil {
		return nil, err
	}

	data2, err := slice.LoadRef()
	if err != nil {
		return nil, err
	}
	target, err := data2.LoadSlice(data2.BitsLeft())
	if err != nil {
		return nil, err
	}

	data3, err := slice.LoadRef()
	if err != nil {
		return nil, err
	}
	payload, err := loadPayload(data3)
	if err != nil {
		return nil, err
	}

	data4, err := slice.LoadRef()
	if err != nil {
		return nil, err
	}
	orderID, err := data4.LoadBigUInt(256)
	if err != nil {
		return nil, err
	}
	mos, err := data4.LoadBigUInt(256)
	if err != nil {
		return nil, err
	}
	token, err := data4.LoadAddr()
	if err != nil {
		return nil, err
	}
	amount, err := data4.LoadBigUInt(128)
	if err != nil {
		return nil, err
	}

	isRelay := false
	if relay.Uint64() == 1 {
		isRelay = true
	}

	oid := [32]byte{}
	copy(oid[:], common.Hex2Bytes(orderID.Text(16)))

	var tokenBytes []byte
	if !token.IsAddrNone() {
		tokenBytes = common.Hex2Bytes(convertToHex(token))
	}

	messageOutEvent := &MessageOutEvent{
		Relay:       isRelay,
		MessageType: uint8(msgType.Uint64()),
		FromChain:   fromChain,
		ToChain:     toChain,
		OrderId:     oid,
		Mos:         common.Hex2Bytes(mos.Text(16)),
		Token:       tokenBytes,
		Initiator:   common.Hex2Bytes(convertToHex(initiator)),
		From:        common.Hex2Bytes(convertToHex(sender)),
		To:          target,
		Amount:      amount,
		GasLimit:    gasLimit,
		SwapData:    payload,
	}

	// todo remove debug log
	fmt.Println("relay: ", messageOutEvent.Relay)
	fmt.Println("msgType: ", messageOutEvent.MessageType)
	fmt.Println("fromChain: ", messageOutEvent.FromChain)
	fmt.Println("toChain: ", messageOutEvent.ToChain)
	fmt.Println("orderID: ", common.Bytes2Hex(messageOutEvent.OrderId[:]))
	fmt.Println("mos: ", common.Bytes2Hex(messageOutEvent.Mos))
	fmt.Println("token: ", common.Bytes2Hex(messageOutEvent.Token))
	fmt.Println("initiator: ", common.Bytes2Hex(messageOutEvent.Initiator))
	fmt.Println("from: ", common.Bytes2Hex(messageOutEvent.From))
	fmt.Println("to: ", common.Bytes2Hex(messageOutEvent.To))
	fmt.Println("amount: ", messageOutEvent.Amount)
	fmt.Println("gasLimit: ", messageOutEvent.GasLimit)
	fmt.Println("payload: ", common.Bytes2Hex(messageOutEvent.SwapData))

	return messageOutEvent, nil
}

func encodeMessageOutEvent(messageOut *MessageOutEvent) ([]byte, error) {
	log.Printf("event: %+v\n", messageOut)
	return messageOutArgs.Pack(messageOut)
}

func encodeProof(addr, topic, data []byte) ([]byte, error) {
	// todo remove debug log
	fmt.Println("addr: ", common.Bytes2Hex(addr))
	fmt.Println("topic: ", common.Bytes2Hex(topic))
	fmt.Println("data: ", common.Bytes2Hex(data))

	return proofArgs.Pack(addr, topic, data)
}

func convertToBytes(addr *address.Address) []byte {
	return append(common.LeftPadBytes(big.NewInt(int64(addr.Workchain())).Bytes(), 1), addr.Data()...)
}

func convertToHex(addr *address.Address) string {
	return fmt.Sprintf("%x", append(common.LeftPadBytes(big.NewInt(int64(addr.Workchain())).Bytes(), 1), addr.Data()...))
}

func loadPayload(slice *cell.Slice) ([]byte, error) {
	payload := make([]byte, 0)
	currentSlice := slice
	for {
		if currentSlice.RefsNum() == 0 {
			break
		}

		part, err := currentSlice.LoadSlice(currentSlice.BitsLeft())
		if err != nil {
			return nil, err
		}
		payload = append(payload, part...)

		nextSlice, err := currentSlice.LoadRef()
		if err != nil {
			return nil, err
		}
		currentSlice = nextSlice
	}
	return payload, nil
}
