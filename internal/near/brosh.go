package near

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"

	"github.com/mr-tron/base58"

	"github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/types/key"
	"github.com/mapprotocol/near-api-go/pkg/types/signature"
)

const (
	Version2         = "V2"
	ValidatorStakeV1 = 0
	ValidatorStakeV2 = 1
)

func Borshify(block client.LightClientBlockView) []byte {
	var (
		buf                bytes.Buffer
		innerLite          bytes.Buffer
		nextBps            bytes.Buffer
		littleEndian       bytes.Buffer
		approvalsAfterNext bytes.Buffer
	)

	buf.Write(MustBase58Decode(block.PrevBlockHash.String()))
	buf.Write(MustBase58Decode(block.NextBlockInnerHash.String()))

	MustToLittleEndian(&littleEndian, block.InnerLite.Height)
	innerLite.Write(littleEndian.Bytes())

	innerLite.Write(MustBase58Decode(block.InnerLite.EpochID.String()))
	innerLite.Write(MustBase58Decode(block.InnerLite.NextEpochId.String()))
	innerLite.Write(MustBase58Decode(block.InnerLite.PrevStateRoot.String()))
	innerLite.Write(MustBase58Decode(block.InnerLite.OutcomeRoot.String()))

	littleEndian.Reset()
	MustToLittleEndian(&littleEndian, block.InnerLite.Timestamp)
	innerLite.Write(littleEndian.Bytes())

	innerLite.Write(MustBase58Decode(block.InnerLite.NextBpHash.String()))
	innerLite.Write(MustBase58Decode(block.InnerLite.BlockMerkleRoot.String()))
	buf.Write(innerLite.Bytes())

	buf.Write(MustBase58Decode(block.InnerRestHash.String()))
	buf.Write([]byte{1})

	littleEndian.Reset()
	MustToLittleEndian(&littleEndian, int64(len(block.NextBps)))
	buf.Write(littleEndian.Next(4))

	for _, bp := range block.NextBps {
		var nextBp bytes.Buffer
		if bp.ValidatorStakeStructVersion == Version2 {
			nextBp.Write([]byte{ValidatorStakeV2})
		} else {
			nextBp.Write([]byte{ValidatorStakeV1})
		}

		littleEndian.Reset()
		MustToLittleEndian(&littleEndian, int64(len(bp.AccountID)))
		nextBp.Write(littleEndian.Next(4))

		nextBp.Write([]byte(bp.AccountID))
		if bp.PublicKey.Type == key.KeyTypeED25519 {
			nextBp.Write([]byte{0})
		} else {
			nextBp.Write([]byte{1})
		}
		nextBp.Write(MustBase58Decode(bp.PublicKey.Value))

		stake, ok := new(big.Int).SetString(bp.Stake.String(), 10)
		if !ok {
			panic(fmt.Sprintf("stake convert to big.Int failed, stake: %s", bp.Stake.String()))
		}
		nextBp.Write(reverse16(stake.Bytes()))
		nextBps.Write(nextBp.Bytes())
	}
	buf.Write(nextBps.Bytes())

	littleEndian.Reset()
	MustToLittleEndian(&littleEndian, int64(len(block.ApprovalsAfterNext)))
	buf.Write(littleEndian.Next(4))
	for _, sign := range block.ApprovalsAfterNext {
		var aan bytes.Buffer
		if sign == nil {
			aan.Write([]byte{0})
		} else {
			aan.Write([]byte{1})
			if sign.Type == signature.SignatureTypeED25519 {
				aan.Write([]byte{0})
			} else {
				aan.Write([]byte{1})
			}
			aan.Write(MustBase58Decode(sign.Value))
		}
		approvalsAfterNext.Write(aan.Bytes())
	}
	buf.Write(approvalsAfterNext.Bytes())

	return buf.Bytes()
}

func reverse16(bs []byte) []byte {
	length := len(bs)
	wbs := make([]byte, length)
	copy(wbs, bs)

	for i := 0; i < len(wbs)/2; i++ {
		wbs[i], wbs[len(wbs)-i-1] = wbs[len(wbs)-i-1], wbs[i]
	}

	fillSize := 16 - length
	if fillSize > 0 {
		for i := 0; i < fillSize; i++ {
			wbs = append(wbs, 0)
		}
	}
	return wbs
}

func MustBase58Decode(str string) []byte {
	dec, err := base58.Decode(str)
	if err != nil {
		panic(err)
	}
	return dec
}

func MustToLittleEndian(w io.Writer, data interface{}) {
	if err := binary.Write(w, binary.LittleEndian, data); err != nil {
		panic(err)
	}
}
