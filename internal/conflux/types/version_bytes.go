package types

import "github.com/pkg/errors"

type VersionByte struct {
	TypeBits uint8
	// current is constant 0, it's different with AddressType defined in address_type.go
	AddressType uint8
	SizeBits    uint8
}

var (
	hashSizeToBits map[uint]uint8 = make(map[uint]uint8)
)

func init() {
	hashSizeToBits[160] = 0
	hashSizeToBits[192] = 1
	hashSizeToBits[224] = 2
	hashSizeToBits[256] = 3
	hashSizeToBits[320] = 4
	hashSizeToBits[384] = 5
	hashSizeToBits[448] = 6
	hashSizeToBits[512] = 7
}

// ToByte returns byte
func (v VersionByte) ToByte() (byte, error) {
	ret := v.TypeBits & 0x80
	ret = ret | v.AddressType<<3
	ret = ret | v.SizeBits
	return ret, nil
}

// NewVersionByte creates version byte by byte
func NewVersionByte(b byte) (vt VersionByte) {
	vt.TypeBits = b >> 7
	vt.AddressType = (b & 0x7f) >> 3
	vt.SizeBits = b & 0x0f
	return
}

func CalcVersionByte(hexAddress []byte) (versionByte VersionByte, err error) {
	versionByte.TypeBits = 0
	versionByte.AddressType = 0
	addrBitsLen := uint(len(hexAddress) * 8)
	versionByte.SizeBits = hashSizeToBits[addrBitsLen]
	if versionByte.SizeBits == 0 && addrBitsLen != 160 {
		return versionByte, errors.Errorf("Invalid hash size %v", addrBitsLen)
	}
	return
}
