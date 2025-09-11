package near

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
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

const (
	ProofDirectionRight = "Right"
	ProofDirectionLeft  = "Left"
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

func BorshifyOutcomeProof(proof client.RpcLightClientExecutionProofResponse) ([]byte, error) {
	var (
		buf           bytes.Buffer
		tmp           bytes.Buffer
		outcomeProof  bytes.Buffer
		outcomeProof2 bytes.Buffer
	)

	// Step1：outComeProof.proof length
	MustToLittleEndian(&tmp, int64(len(proof.OutcomeProof.Proof)))
	outcomeProof.Write(tmp.Next(4))
	tmp.Reset()
	// Step2：outComeProof.proof
	for _, p := range proof.OutcomeProof.Proof {
		tmp.Write(MustBase58Decode(p.Hash.String()))
		if p.Direction == ProofDirectionRight {
			tmp.Write([]byte{1})
		} else {
			tmp.Write([]byte{0})
		}
		outcomeProof.Write(tmp.Bytes())
		tmp.Reset()
	}
	buf.Write(outcomeProof.Bytes())
	buf.Write(MustBase58Decode(proof.OutcomeProof.BlockHash.String()))
	buf.Write(MustBase58Decode(proof.OutcomeProof.ID.String()))
	// step3: outComeProof.outCome.logs
	// 3.1 length
	MustToLittleEndian(&tmp, int64(len(proof.OutcomeProof.Outcome.Logs)))
	outcomeProof2.Write(tmp.Next(4))
	tmp.Reset()
	// 3.2 logs
	logBuf := bytes.Buffer{}
	for _, l := range proof.OutcomeProof.Outcome.Logs {
		var lb bytes.Buffer
		MustToLittleEndian(&tmp, int64(len(l)))
		lb.Write(tmp.Next(4))
		tmp.Reset()
		lb.Write([]byte(l))
		logBuf.Write(lb.Bytes())
	}
	outcomeProof2.Write(logBuf.Bytes())
	// step4: outComeProof.outCome.receiptIDs
	MustToLittleEndian(&tmp, int64(len(proof.OutcomeProof.Outcome.ReceiptIDs)))
	outcomeProof2.Write(tmp.Next(4))
	tmp.Reset()
	// step:4.1
	receiptIDs := bytes.Buffer{}
	for _, rId := range proof.OutcomeProof.Outcome.ReceiptIDs {
		var rIdBuf bytes.Buffer
		rIdBuf.Write(MustBase58Decode(rId.String()))
		receiptIDs.Write(rIdBuf.Bytes())
	}
	outcomeProof2.Write(receiptIDs.Bytes())
	// step:4.2
	MustToLittleEndian(&tmp, proof.OutcomeProof.Outcome.GasBurnt)
	outcomeProof2.Write(tmp.Next(8))
	// step:4.3
	MustToLittleEndian(&tmp, proof.OutcomeProof.Outcome.TokensBurnt)
	outcomeProof2.Write(tmp.Next(16))
	tmp.Reset()
	// step:4.4
	MustToLittleEndian(&tmp, int64(len(proof.OutcomeProof.Outcome.ExecutorID)))
	outcomeProof2.Write(tmp.Next(4))
	tmp.Reset()
	// step:4.5
	outcomeProof2.Write([]byte(proof.OutcomeProof.Outcome.ExecutorID))
	// step:4.6
	statusByte, err := resolveStatus(&proof.OutcomeProof.Outcome.Status)
	if err != nil {
		return nil, err
	}
	outcomeProof2.Write(statusByte)
	// step5 outcomeRootProof
	MustToLittleEndian(&tmp, int64(len(proof.OutcomeRootProof)))
	outcomeProof2.Write(tmp.Next(4))
	tmp.Reset()
	// step5.1 outcome_root_proof
	rootProof := bytes.Buffer{}
	for _, p := range proof.OutcomeRootProof {
		var pt bytes.Buffer
		pt.Write(MustBase58Decode(p.Hash.String()))
		if p.Direction == ProofDirectionRight {
			pt.Write([]byte{1})
		} else {
			pt.Write([]byte{0})
		}
		rootProof.Write(pt.Bytes())
	}
	outcomeProof2.Write(rootProof.Bytes())
	// step6 block_header_lite
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.PrevBlockHash.String()))
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerRestHash.String()))
	MustToLittleEndian(&tmp, proof.BlockHeaderLite.InnerLite.Height)
	outcomeProof2.Write(tmp.Next(8))
	tmp.Reset()
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.EpochID.String()))
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.NextEpochId.String()))
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.PrevStateRoot.String()))
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.OutcomeRoot.String()))
	MustToLittleEndian(&tmp, proof.BlockHeaderLite.InnerLite.Timestamp)
	outcomeProof2.Write(tmp.Next(8))
	tmp.Reset()
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.NextBpHash.String()))
	outcomeProof2.Write(MustBase58Decode(proof.BlockHeaderLite.InnerLite.BlockMerkleRoot.String()))
	// step7 blockProof
	MustToLittleEndian(&tmp, int64(len(proof.BlockProof)))
	outcomeProof2.Write(tmp.Next(4))
	tmp.Reset()
	var bpBuf bytes.Buffer
	for _, bp := range proof.BlockProof {
		var bpb bytes.Buffer
		bpb.Write(MustBase58Decode(bp.Hash.String()))
		if bp.Direction == ProofDirectionRight {
			bpb.Write([]byte{1})
		} else {
			bpb.Write([]byte{0})
		}
		bpBuf.Write(bpb.Bytes())
	}
	outcomeProof2.Write(bpBuf.Bytes())
	buf.Write(outcomeProof2.Bytes())
	return buf.Bytes(), nil
}

func resolveStatus(status *client.TransactionStatus) ([]byte, error) {
	data, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	var ret bytes.Buffer
	if v, ok := m["SuccessValue"]; ok {
		// step1
		ret.Write([]byte{2})
		// step2
		res, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return nil, err
		}
		var tmp bytes.Buffer
		MustToLittleEndian(&tmp, int64(len(res)))
		ret.Write(tmp.Next(4))
		// step3
		ret.Write(res)
	} else if v, ok := m["SuccessReceiptId"]; ok {
		ret.Write([]byte{3})
		ret.Write(MustBase58Decode(v.(string)))
	} else {
		return nil, errors.New("transaction status not supported")
	}

	return ret.Bytes(), nil
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
		fmt.Println("binary.Write failed:", err)
	}
}
