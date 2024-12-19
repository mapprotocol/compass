package ton

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"math"
	"math/big"
	"strings"
)

type Signature struct {
	V uint64
	R *big.Int
	S *big.Int
}

func GenerateMessageInCell(
	hash common.Hash,
	expectedAddress common.Address,
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
		signsCell = signsCell.MustStoreRef(cell.BeginCell().MustStoreUInt(signs[i].V, 8).MustStoreBigUInt(signs[i].R, 256).MustStoreBigUInt(signs[i].S, 256).EndCell())
	}

	msg, err := EncodeMessage(message)
	if err != nil {
		return nil, err
	}

	return cell.BeginCell().
		MustStoreUInt(0xd5f86120, 32).
		MustStoreUInt(0, 64).
		MustStoreBigUInt(hash.Big(), 256).
		//MustStoreBigUInt(expectedAddress.Big(), 160).
		MustStoreBigUInt(new(big.Int).SetBytes(expectedAddress[:]), 160).
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
				//MustStoreBigUInt(addr.Big(), 256).
				MustStoreBigUInt(new(big.Int).SetBytes(addr[:]), 256).
				MustStoreRef(
					cell.BeginCell().
						MustStoreBigUInt(topics[0].Big(), 256).
						MustStoreBigUInt(topics[1].Big(), 256).
						MustStoreBigUInt(topics[2].Big(), 256).
						EndCell()).
				MustStoreRef(msg).
				EndCell(),
		).EndCell(), nil

}

func EncodeMessage(data []byte) (*cell.Cell, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("data too short, minimum 64 bytes required")
	}

	// Extract metadata and data positions
	offset := data[:32]
	length := data[32:64]
	pos := uint(64)

	// Build metadata cell
	metadataCell := cell.BeginCell().
		MustStoreSlice(offset, uint(len(offset))*8).
		MustStoreSlice(length, uint(len(length))*8).
		EndCell()

	// Read header fields
	version := data[pos]
	pos++
	relay := data[pos]
	pos++
	tokenLen := uint(data[pos])
	pos++
	mosLen := uint(data[pos])
	pos++
	fromLen := uint(data[pos])
	pos++
	toLen := uint(data[pos])
	pos++
	payloadLen := uint(data[pos])<<8 | uint(data[pos+1])
	pos += 2

	// Read reserved and token amount
	reserved := data[pos : pos+8]
	pos += 8
	tokenAmount := data[pos : pos+16]
	pos += 16

	// Read addresses
	tokenAddr := data[pos : pos+tokenLen]
	pos += tokenLen
	mosTarget := data[pos : pos+mosLen]
	pos += mosLen
	fromAddr := data[pos : pos+fromLen]
	pos += fromLen
	toAddr := data[pos : pos+toLen]
	pos += toLen
	payload := data[pos : pos+payloadLen]

	// Build header cell
	headerCell := cell.BeginCell().
		MustStoreUInt(uint64(version), 8).
		MustStoreUInt(uint64(relay), 8).
		MustStoreUInt(uint64(tokenLen), 8).
		MustStoreUInt(uint64(mosLen), 8).
		MustStoreUInt(uint64(fromLen), 8).
		MustStoreUInt(uint64(toLen), 8).
		MustStoreUInt(uint64(payloadLen), 16).
		MustStoreSlice(reserved, uint(len(reserved))*8).
		MustStoreSlice(tokenAmount, uint(len(tokenAmount))*8).
		EndCell()

	// Build addresses cells
	tokenMosCell := cell.BeginCell().
		MustStoreSlice(tokenAddr, uint(len(tokenAddr))*8).
		MustStoreSlice(mosTarget, uint(len(mosTarget))*8).
		EndCell()

	fromToCell := cell.BeginCell().
		MustStoreSlice(fromAddr, uint(len(fromAddr))*8).
		MustStoreSlice(toAddr, uint(len(toAddr))*8).
		EndCell()

	// Build payload cell
	payloadCell, err := EncodePayload(payload)
	if err != nil {
		return nil, err
	}

	// Link all cells together
	metadataAndHeader := cell.BeginCell().
		MustStoreRef(metadataCell).
		MustStoreRef(headerCell).
		EndCell()

	return cell.BeginCell().
		MustStoreRef(metadataAndHeader).
		MustStoreRef(tokenMosCell).
		MustStoreRef(fromToCell).
		MustStoreRef(payloadCell).
		EndCell(), nil
}

const (
	maxBits         = 768
	maxBytesPerCell = maxBits / 8 // ~127 bytes
)

func bytesToBinary(data []byte) string {
	var binary strings.Builder
	for _, b := range data {
		binary.WriteString(fmt.Sprintf("%08b", b))
	}
	return binary.String()
}

func binaryToBytes(binary string) []byte {
	// Pad binary string to multiple of 8
	padding := len(binary) % 8
	if padding != 0 {
		binary = binary + strings.Repeat("0", 8-padding)
	}

	result := make([]byte, len(binary)/8)
	for i := 0; i < len(binary); i += 8 {
		end := i + 8
		if end > len(binary) {
			end = len(binary)
		}
		chunk := binary[i:end]

		var val byte
		for j, bit := range chunk {
			if bit == '1' {
				val |= 1 << (7 - j)
			}
		}
		result[i/8] = val
	}
	return result
}

func EncodePayload(data []byte) (*cell.Cell, error) {
	binaryStr := bytesToBinary(data)

	var cells []*cell.Builder
	position := 0

	for position < len(binaryStr) {
		builder := cell.BeginCell()
		end := position + maxBits
		if end > len(binaryStr) {
			end = len(binaryStr)
		}
		bitsForCurrentCell := binaryStr[position:end]

		for _, bit := range bitsForCurrentCell {
			if bit == '1' {
				builder.StoreBoolBit(true)
			} else {
				builder.StoreBoolBit(false)
			}
		}

		cells = append(cells, builder)
		position = end
	}

	lastCell := cells[len(cells)-1].EndCell()
	for i := len(cells) - 2; i >= 0; i-- {
		cells[i].StoreRef(lastCell)
		lastCell = cells[i].EndCell()
	}

	return lastCell, nil
}

func DecodePayload(rootCell *cell.Cell) ([]byte, error) {
	var binaryResult strings.Builder
	currentCell := rootCell

	for {
		slice := currentCell.BeginParse()

		for slice.BitsLeft() > 0 {
			bit, err := slice.LoadBoolBit()
			if err != nil {
				return nil, fmt.Errorf("error loading bit: %w", err)
			}
			if bit {
				binaryResult.WriteString("1")
			} else {
				binaryResult.WriteString("0")
			}
		}

		if slice.RefsNum() == 0 {
			break
		}

		nextCell, err := slice.LoadRefCell()
		if err != nil {
			return nil, fmt.Errorf("error loading next cell: %w", err)
		}
		currentCell = nextCell
	}

	return binaryToBytes(binaryResult.String()), nil
}

func CalculateCellCount(data []byte) int {
	return int(math.Ceil(float64(len(data)) / float64(maxBytesPerCell)))
}
