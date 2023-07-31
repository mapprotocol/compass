package types

import "github.com/pkg/errors"

type Checksum [8]byte

// CalcChecksum calculates checksum by network type and body
func CalcChecksum(nt NetworkType, body Body) (c Checksum, err error) {
	var lower5bitsNettype []byte
	for _, v := range nt.String() {
		lower5bitsNettype = append(lower5bitsNettype, byte(v)&0x1f)
	}
	separator := byte(0)
	payload5Bits := body
	template := [8]byte{}

	checksumInput := append(lower5bitsNettype, separator)
	checksumInput = append(checksumInput, payload5Bits[:]...)
	checksumInput = append(checksumInput, template[:]...)

	// fmt.Printf("checksumInput:%x\n", checksumInput)

	uint64Chc := polymod(checksumInput)
	// fmt.Printf("uint64Chc:%x\n", uint64Chc)

	low40BitsChc := uint64ToBytes(uint64Chc)[3:]
	// fmt.Printf("low40BitsChc of %x:%x\n", uint64ToBytes(uint64Chc), low40BitsChc)

	checksumIn5Bits, err := convert(low40BitsChc, 8, 5)
	// fmt.Printf("low40BitsChcIn5Bits:%x\n", checksumIn5Bits)

	if err != nil {
		err = errors.Wrapf(err, "failed to convert %v from 8 to 5 bits", low40BitsChc)
		return
	}
	copy(c[:], checksumIn5Bits)
	return
}

// String returns base32 string of checksum according to CIP-37
func (c Checksum) String() string {
	return bits5sToString(c[:])
}

func polymod(v []byte) uint64 {
	c := uint64(1)
	for _, d := range v {
		c0 := byte(c >> 35)
		c = ((c & 0x07ffffffff) << 5) ^ uint64(d)
		if c0&0x01 != 0 {
			c ^= 0x98f2bc8e61
		}
		if c0&0x02 != 0 {
			c ^= 0x79b76d99e2
		}
		if c0&0x04 != 0 {
			c ^= 0xf33e5fb3c4
		}
		if c0&0x08 != 0 {
			c ^= 0xae2eabe2a8
		}
		if c0&0x10 != 0 {
			c ^= 0x1e4f43e470
		}
	}
	return c ^ 1
}

func uint64ToBytes(num uint64) []byte {
	r := make([]byte, 8)
	for i := 0; i < 8; i++ {
		r[7-i] = byte(num >> uint(i*8))
	}
	return r
}
