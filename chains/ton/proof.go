package ton

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
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

type Signature struct {
	V uint64
	R *big.Int
	S *big.Int
}

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

func GenerateMessageInCell(
	hash common.Hash,
	signs []*Signature,
	receiptRoot common.Hash,
	version common.Hash,
	blockNum int64,
	chainId int64,
	addr common.Address,
	topics []common.Hash,
	message []byte,
) (*cell.Cell, error) {

	signsCell := cell.BeginCell()
	for i := 0; i < len(signs); i++ {
		signsCell = signsCell.MustStoreRef(cell.BeginCell().MustStoreUInt(signs[i].V, 8).MustStoreBigUInt(signs[i].R, 256).EndCell())
	}

	return cell.BeginCell().
		MustStoreUInt(0xd5f86120, 32).
		MustStoreUInt(0, 64).
		MustStoreBigUInt(hash.Big(), 256).
		MustStoreUInt(uint64(len(signs)), 8).
		MustStoreRef(signsCell.EndCell()).
		MustStoreRef(
			cell.BeginCell().
				MustStoreBigUInt(receiptRoot.Big(), 256).
				MustStoreBigUInt(version.Big(), 256).
				MustStoreBigUInt(big.NewInt(blockNum), 256).
				MustStoreInt(chainId, 64).
				EndCell()).
		MustStoreRef(
			cell.BeginCell().
				MustStoreBigUInt(new(big.Int).SetBytes(addr[:]), 256).
				MustStoreRef(
					cell.BeginCell().
						MustStoreBigUInt(topics[0].Big(), 256).
						MustStoreBigUInt(topics[1].Big(), 256).
						MustStoreBigUInt(topics[2].Big(), 256).
						EndCell()).
				MustStoreSlice(message, uint(len(message))).
				EndCell(),
		).EndCell(), nil

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
	mos, err := data4.LoadAddr()
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

	fmt.Println("relay: ", relay)
	fmt.Println("msgType: ", msgType)
	fmt.Println("fromChain: ", fromChain)
	fmt.Println("toChain: ", toChain)
	fmt.Println("gasLimit: ", gasLimit)
	fmt.Println("initiator: ", initiator)
	fmt.Println("sender: ", sender)
	fmt.Println("target: ", "0x"+common.Bytes2Hex(target))
	fmt.Println("payload: ", payload)
	fmt.Println("orderID: ", orderID)
	fmt.Println("mos: ", mos.Bounce(false))
	fmt.Println("token: ", token)
	fmt.Println("amount: ", amount)

	isRelay := false
	if relay.Uint64() == 1 {
		isRelay = true
	}

	oid := [32]byte{}
	copy(oid[:], common.Hex2Bytes(orderID.Text(16)))

	messageOutEvent := &MessageOutEvent{
		Relay: isRelay,
		//MessageType: uint8(msgType.Uint64()),
		MessageType: 1, // todo
		FromChain:   fromChain,
		ToChain:     toChain,
		OrderId:     oid,
		Mos:         common.Hex2Bytes(convertToHex(mos)),
		//Token:       common.Hex2Bytes(convertToHex(token)),
		Token:     []byte{}, // todo
		Initiator: common.Hex2Bytes(convertToHex(initiator)),
		From:      common.Hex2Bytes(convertToHex(sender)),
		To:        target,
		Amount:    big.NewInt(0),
		GasLimit:  big.NewInt(200000000),
		SwapData:  payload, // todo payload
	}

	return messageOutEvent, nil
}

func encodeMessageOutEvent(messageOut *MessageOutEvent) ([]byte, error) {
	log.Printf("event: %+v\n", messageOut)
	return messageOutArgs.Pack(messageOut)
}

func encodeProof(addr, topic, data []byte) ([]byte, error) {
	fmt.Println("============================== addr: ", common.Bytes2Hex(addr))
	fmt.Println("============================== topic: ", common.Bytes2Hex(topic))
	fmt.Println("============================== data: ", common.Bytes2Hex(data))

	return proofArgs.Pack(addr, topic, data)
}

func convertToBytes(addr *address.Address) []byte {
	return append(common.LeftPadBytes(big.NewInt(1).Bytes(), 1), addr.Data()...)
}

func convertToHex(addr *address.Address) string {
	return fmt.Sprintf("%x", append(common.LeftPadBytes(big.NewInt(1).Bytes(), 1), addr.Data()...))
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
