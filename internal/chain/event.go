package chain

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	EventMessageRelay = "MessageRelay"
)

var (
	messageRelayABI, _ = abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"bytes32","name":"orderId","type":"bytes32"},{"indexed":true,"internalType":"uint256","name":"chainAndGasLimit","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"payload","type":"bytes"}],"name":"MessageRelay","type":"event"}]`))
	relayDecodeAbi, _  = abi.JSON(strings.NewReader(`[{"inputs":[{"internalType":"bytes","name":"","type":"bytes"}],"name":"relayDecode","outputs":[{"internalType":"bytes32","name":"header","type":"bytes32"},{"internalType":"address","name":"mos","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"address","name":"to","type":"address"},{"internalType":"bytes","name":"from","type":"bytes"},{"internalType":"bytes","name":"swapData","type":"bytes"}],"stateMutability":"pure","type":"function"}]`))
)

type Swap struct {
	ToToken   []byte
	Receiver  []byte
	MinAmount *big.Int
}

type MessageOutTokens struct {
	Token    common.Address
	DstToken []byte
}

type Relay struct {
	Version     *big.Int
	MessageType *big.Int
	OrderId     [32]byte // MessageRelay.OrderId
	SrcChain    *big.Int // MessageRelay.ChainAndGasLimit 0:8
	Sender      string   // MessageRelayPayload.From
	DstChain    *big.Int // MessageRelay.ChainAndGasLimit 8:16
	DstToken    []byte   // MessageRelayPayload.TokenAddress
	OutAmount   *big.Int // MessageRelayPayload.TokenAmount
	Receiver    []byte   // MessageRelayPayload.To
	Payload     []byte
	Swap        *Swap
}

type Contract struct {
	abi abi.ABI
}

func NewContract(abi abi.ABI) *Contract {
	return &Contract{abi: abi}
}

func (c *Contract) UnpackLog(ret interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.abi.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.abi.UnpackIntoInterface(ret, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(ret, indexed, log.Topics[1:])
}

type MessageRelay struct {
	OrderId          [32]byte
	ChainAndGasLimit *big.Int // fromChain (8 bytes) | toChain (8 bytes) | reserved (8 bytes) | gasLimit (8 bytes)
	Payload          []byte   // MessageRelayPayload
}

func UnpackMessageRelay(log types.Log) (*MessageRelay, error) {
	ret := &MessageRelay{}
	if err := NewContract(messageRelayABI).UnpackLog(ret, EventMessageRelay, log); err != nil {
		return nil, err
	}
	return ret, nil
}

func DecodeMessageRelay(log *types.Log, targetEvm bool) (string, string, error) {
	fmt.Println("MessageRelay.txHash:", log.TxHash.Hex())
	fmt.Println("MessageRelay.orderId:", log.Topics[1].Hex())
	fmt.Println("MessageRelay.targetEvm:", targetEvm)

	/* --------------------------------
	   decode(["bytes"], event.data)
	----------------------------------*/
	bytesArg := abi.Arguments{
		{Type: mustType("bytes")},
	}

	unpacked, err := bytesArg.Unpack(log.Data)
	if err != nil {
		return "", "", err
	}

	chainAndGasLimit := unpacked[0].([]byte)
	/* ============================================================
	   targetEvm == true → ABI 解码
	============================================================ */
	if targetEvm {
		args := abi.Arguments{
			{Type: mustType("bytes32")},
			{Type: mustType("address")},
			{Type: mustType("address")},
			{Type: mustType("uint256")},
			{Type: mustType("address")},
			{Type: mustType("bytes")},
			{Type: mustType("bytes")},
		}
		values, err := args.Unpack(chainAndGasLimit)
		if err != nil {
			return "", "", err
		}

		header := values[0].([32]byte)
		mos := values[1].(common.Address)
		token := values[2].(common.Address)
		amount := values[3].(*big.Int)
		to := values[4].(common.Address)
		from := values[5].([]byte)
		swapData := values[6].([]byte)

		fmt.Printf("MessageRelay.header: 0x%x\n", header)
		fmt.Println("MessageRelay.mos:", mos.Hex())
		fmt.Println("MessageRelay.token:", token.Hex())
		fmt.Println("MessageRelay.amount:", amount.String())
		fmt.Println("MessageRelay.to:", to.Hex())
		fmt.Printf("MessageRelay.from: 0x%x\n", from)

		if len(swapData) == 0 {
			fmt.Println("MessageRelay.swapData:", swapData)
			return fmt.Sprintf("0x%x", from), to.Hex(), nil
		}

		fmt.Println("<-----------------------------MessageRelay swapAndCall----------------------------------------------------->")
		return fmt.Sprintf("0x%x", from), to.Hex(), nil
	}

	/* ============================================================
	   targetEvm == false → 手动解析 hex
	============================================================ */

	hexStr := hex.EncodeToString(chainAndGasLimit)

	readBig := func(start, end int) *big.Int {
		v, _ := new(big.Int).SetString(hexStr[start:end], 16)
		return v
	}

	version := readBig(0, 4)
	fmt.Println("MessageRelay.version:", version)

	messageType := readBig(4, 6)
	fmt.Println("MessageRelay.messageType:", messageType)

	tokenLen := readBig(6, 8)
	fmt.Println("MessageRelay.tokenLen:", tokenLen)

	mosLen := readBig(8, 10)
	fmt.Println("MessageRelay.mosLen:", mosLen)

	fromLen := readBig(10, 12)
	fmt.Println("MessageRelay.fromLen:", fromLen)

	toLen := readBig(12, 14)
	fmt.Println("MessageRelay.toLen:", toLen)

	payloadLen := readBig(14, 18)
	fmt.Println("MessageRelay.payloadLen:", payloadLen)

	tokenAmount := readBig(34, 66)
	fmt.Println("MessageRelay.tokenAmount:", tokenAmount)

	start := 66
	end := start + int(tokenLen.Int64())*2
	tokenAddress := "0x" + hexStr[start:end]
	fmt.Println("MessageRelay.tokenAddress:", tokenAddress)

	start = end
	end = start + int(mosLen.Int64())*2
	mos := "0x" + hexStr[start:end]
	fmt.Println("MessageRelay.mos:", mos)

	start = end
	end = start + int(fromLen.Int64())*2
	from := "0x" + hexStr[start:end]
	fmt.Println("MessageRelay.from:", from)

	start = end
	end = start + int(toLen.Int64())*2
	to := "0x" + hexStr[start:end]
	fmt.Println("MessageRelay.to:", to)

	start = end
	payload := "0x" + hexStr[start:]
	fmt.Println("MessageRelay.payload:", payload)

	return from, to, nil
}

func DecodeMessageRelayTokens(log *types.Log) (*MessageOutTokens, error) {
	data, err := unpackLogBytes(log.Data)
	if err != nil {
		return nil, err
	}

	args := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("address")},
		{Type: mustType("address")},
		{Type: mustType("uint256")},
		{Type: mustType("address")},
		{Type: mustType("bytes")},
		{Type: mustType("bytes")},
	}
	unpacked, err := args.Unpack(data)
	if err != nil {
		return decodeNonEVMMessageRelayTokens(data)
	}

	token := unpacked[2].(common.Address)
	swapData := unpacked[6].([]byte)
	if len(swapData) == 0 {
		return &MessageOutTokens{Token: token}, nil
	}

	dstToken, err := decodeSwapDstToken(swapData)
	if err != nil {
		return &MessageOutTokens{Token: token}, nil
	}
	return &MessageOutTokens{Token: token, DstToken: dstToken}, nil
}

func decodeNonEVMMessageRelayTokens(data []byte) (*MessageOutTokens, error) {
	const nonEVMRelayHeaderLen = 33
	if len(data) < nonEVMRelayHeaderLen {
		return nil, fmt.Errorf("message relay payload too short: %d", len(data))
	}
	tokenLen := int(data[3])
	if tokenLen == 0 {
		return &MessageOutTokens{}, nil
	}
	if len(data) < nonEVMRelayHeaderLen+tokenLen {
		return nil, fmt.Errorf("message relay token out of bounds: payload=%d tokenLen=%d", len(data), tokenLen)
	}
	tokenBytes := data[nonEVMRelayHeaderLen : nonEVMRelayHeaderLen+tokenLen]
	return &MessageOutTokens{Token: addressFromTokenBytes(tokenBytes)}, nil
}

func addressFromTokenBytes(token []byte) common.Address {
	switch {
	case len(token) == common.AddressLength:
		return common.BytesToAddress(token)
	case len(token) == common.HashLength:
		for _, b := range token[:common.HashLength-common.AddressLength] {
			if b != 0 {
				return common.Address{}
			}
		}
		return common.BytesToAddress(token[common.HashLength-common.AddressLength:])
	default:
		return common.Address{}
	}
}

func DecodeMessageOutTokens(log *types.Log) (*MessageOutTokens, error) {
	data, err := unpackLogBytes(log.Data)
	if err != nil {
		return nil, err
	}

	unpacked2, err := messageOutArgs(false).Unpack(data)
	if err != nil {
		unpacked2, err = messageOutArgs(true).Unpack(data)
		if err != nil {
			return nil, err
		}
	}

	token := unpacked2[2].(common.Address)
	_, _, swapData := parseMessageOutTail(unpacked2)
	if len(swapData) <= 0 {
		return &MessageOutTokens{Token: token}, nil
	}

	dstToken, err := decodeSwapDstToken(swapData)
	if err != nil {
		return &MessageOutTokens{Token: token}, nil
	}

	return &MessageOutTokens{Token: token, DstToken: dstToken}, nil
}

func DecodeMessageOut(log *types.Log) (*MessageOutTokens, error) {
	data, err := unpackLogBytes(log.Data)
	if err != nil {
		return nil, err
	}

	unpacked2, err := messageOutArgs(false).Unpack(data)
	if err != nil {
		unpacked2, err = messageOutArgs(true).Unpack(data)
		if err != nil {
			return nil, err
		}
	}

	header := unpacked2[0].([32]byte)
	mos := unpacked2[1].(common.Address)
	token := unpacked2[2].(common.Address)
	amount := unpacked2[3].(*big.Int)
	initiator := unpacked2[4].(common.Address)
	from, to, swapData := parseMessageOutTail(unpacked2)

	fmt.Printf("MessageOut.tx: %s\n", log.TxHash.Hex())
	fmt.Printf("MessageOut.header: 0x%x\n", header)
	relay := header[24] == 0x01
	fmt.Println("MessageOut.relay:", relay)

	fmt.Println("MessageOut.mos:", mos.Hex())
	fmt.Println("MessageOut.token:", token.Hex())
	fmt.Println("MessageOut.amount:", amount.String())
	fmt.Println("MessageOut.initiator:", initiator.Hex())
	fmt.Println("MessageOut.from:", from.Hex())
	fmt.Printf("MessageOut.to: 0x%x\n", to)

	if len(swapData) <= 0 {
		fmt.Println("MessageOut.swapData:", swapData)
		return &MessageOutTokens{Token: token}, nil
	}

	dstToken, err := decodeSwapDstToken(swapData)
	if err != nil {
		return &MessageOutTokens{Token: token}, nil
	}

	return &MessageOutTokens{Token: token, DstToken: dstToken}, nil
}

func unpackLogBytes(data []byte) ([]byte, error) {
	bytesArg := abi.Arguments{
		{
			Type: abi.Type{
				T: abi.BytesTy,
			},
		},
	}

	unpacked1, err := bytesArg.Unpack(data)
	if err != nil {
		return nil, err
	}

	return unpacked1[0].([]byte), nil
}

func messageOutArgs(withFromAddress bool) abi.Arguments {
	args := abi.Arguments{
		{Type: mustType("bytes32")},
		{Type: mustType("address")},
		{Type: mustType("address")},
		{Type: mustType("uint256")},
		{Type: mustType("address")},
	}
	if withFromAddress {
		args = append(args, abi.Argument{Type: mustType("address")})
	}
	return append(args,
		abi.Argument{Type: mustType("bytes")},
		abi.Argument{Type: mustType("bytes")},
	)
}

func parseMessageOutTail(unpacked []interface{}) (common.Address, []byte, []byte) {
	if len(unpacked) == 8 {
		return unpacked[5].(common.Address), unpacked[6].([]byte), unpacked[7].([]byte)
	}
	from := common.Address{}
	fromBytes := unpacked[5].([]byte)
	if len(fromBytes) == common.AddressLength {
		from = common.BytesToAddress(fromBytes)
	}
	return from, unpacked[5].([]byte), unpacked[6].([]byte)
}

func decodeSwapDstToken(swapData []byte) ([]byte, error) {
	args := abi.Arguments{
		{Type: mustType("bool")},
		{Type: mustType("uint256")},
		{Type: mustType("bytes")},
		{Type: mustType("bytes")},
		{Type: mustType("uint256")},
		{Type: mustType("bytes")},
	}
	unpacked, err := args.Unpack(swapData)
	if err != nil {
		return nil, err
	}
	dstToken, ok := unpacked[2].([]byte)
	if !ok {
		return nil, fmt.Errorf("swapData dstToken type mismatch")
	}
	return dstToken, nil
}

func mustType(t string) abi.Type {
	ty, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return ty
}
