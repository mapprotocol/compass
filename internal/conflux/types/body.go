package types

import (
	"github.com/pkg/errors"
	"strings"
)

var (
	ErrorBodyLen = errors.New("Body length must be 34")
)

type Body [34]byte

// NewBodyByString creates body by base32 string which contains version byte and hex address
func NewBodyByString(base32Str string) (body Body, err error) {
	if len(base32Str) != 34 {
		return body, ErrorBodyLen
	}

	for i, v := range base32Str {
		index, ok := alphabetToIndexMap[v]
		if !ok {
			err = errors.New("invalid base32 string for body")
		}
		body[i] = index
	}

	return
}

// NewBodyByHexAddress convert concat of version type and hex address to 5 bits slice
func NewBodyByHexAddress(vrsByte VersionByte, hexAddress []byte) (b Body, err error) {
	vb, err := vrsByte.ToByte()
	if err != nil {
		err = errors.Wrapf(err, "failed to encode version type %#v", vrsByte)
		return
	}
	concatenate := append([]byte{vb}, hexAddress[:]...)
	bits5, err := convert(concatenate, 8, 5)
	if err != nil {
		err = errors.Wrapf(err, "failed to convert %x from 8 to 5 bits array", concatenate)
		return
	}
	// b = bits5
	copy(b[:], bits5[:])
	return
}

// ToHexAddress decode bits5 array to version byte and hex address
func (b Body) ToHexAddress() (vrsType VersionByte, hexAddress []byte, err error) {
	if len(b) != 34 {
		err = errors.New("invalid base32 body, body need be 34 bytes")
		return
	}

	val, err := convert(b[:], 5, 8)
	vrsType = NewVersionByte(val[0])
	hexAddress = val[1:]
	return
}

// String return base32 string
func (b Body) String() string {
	return bits5sToString(b[:])
}

const (
	alphabet = "abcdefghjkmnprstuvwxyz0123456789"
)

var (
	alphabetToIndexMap map[rune]byte = make(map[rune]byte)
)

func init() {
	for i, v := range alphabet {
		alphabetToIndexMap[v] = byte(i)
	}
}

func convert(data []byte, inbits uint, outbits uint) ([]byte, error) {
	// fmt.Printf("convert %b from %v bits to %v bits\n", data, inbits, outbits)
	// only support bits length<=8
	if inbits > 8 || outbits > 8 {
		return nil, errors.New("only support bits length<=8")
	}

	accBits := uint(0) //accumulate bit length
	acc := uint16(0)   //accumulate value
	var ret []byte
	for _, d := range data {
		acc = acc<<uint16(inbits) | uint16(d)
		// fmt.Printf("acc1: %b\n", acc)
		accBits += inbits
		for accBits >= outbits {
			val := byte(acc >> uint16(accBits-outbits))
			// fmt.Printf("5bits val:%v\n", val)
			ret = append(ret, val)
			// fmt.Printf("ret: %b\n", ret)
			acc = acc & uint16(1<<(accBits-outbits)-1)
			// fmt.Printf("acc2: %b\n", acc)
			accBits -= outbits
		}
	}
	// if acc > 0 || accBits > 0 {
	if accBits > 0 && (inbits > outbits) {
		ret = append(ret, byte(acc<<uint16(outbits-accBits)))
	}
	// fmt.Printf("ret %b\n", ret)
	return ret, nil
}

func bits5sToString(dataInBits5 []byte) string {
	sb := strings.Builder{}
	for _, v := range dataInBits5 {
		sb.WriteRune(rune(alphabet[v]))
	}
	return sb.String()
}
