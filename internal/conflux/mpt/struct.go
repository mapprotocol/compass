package mpt

import "math/big"

type TypesReceiptProof struct {
	Headers      [][]byte
	BlockIndex   []byte
	BlockProof   []ProofLibProofNode
	ReceiptsRoot [32]byte
	Index        []byte
	Receipt      []byte
	ReceiptProof []ProofLibProofNode
}

type ProofLibProofNode struct {
	Path     ProofLibNibblePath
	Children [16][32]byte
	Value    []byte
}

type ProofLibNibblePath struct {
	Nibbles [32]byte
	Start   *big.Int
	End     *big.Int
}
