package mapprotocol

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/msg"
)

const (
	MethodVerifyProofData        = "verifyProofData"
	MethodUpdateBlockHeader      = "updateBlockHeader"
	MethodVerifiableHeaderRange  = "verifiableHeaderRange"
	MethodOfHeaderHeight         = "headerHeight"
	MethodOfTransferIn           = "transferIn"
	MethodOfDepositIn            = "depositIn"
	MethodOfSwapIn               = "swapIn"
	MethodOfOrderList            = "orderList"
	MethodOfIsUsedEvent          = "is_used_event"
	MethodOfGetBytes             = "getBytes"
	MethodOfGetFinalBytes        = "getFinalBytes"
	MethodOfGetHeadersBytes      = "getHeadersBytes"
	MethodOfGetBlockHeadersBytes = "getBlockHeaderBytes"
	MethodOfGetUpdatesBytes      = "getUpdateBytes"
	MethodUpdateLightClient      = "updateLightClient"
	MethodClientState            = "clientState"
	MethodClientStateAnalysis    = "clientStateAnalysis"
	MethodOfState                = "state"
	MethodOfNearestPivot         = "nearestPivot"
	MethodOFinalizedState        = "finalizedState"
	MethodOfVerifyReceiptProof   = "verifyReceiptProof"
	MethodOfOrderStatus          = "getOrderStatus"
	MethodOfPropose              = "propose"
	MethodOfVerifyAndStore       = "swapInVerify"
	MethodOfSwapInVerified       = "swapInVerified"
	EventOfSwapInVerified        = "mapSwapInVerified"
	MethodOfTransferInWithIndex  = "transferInWithIndex"
	MethodOfSwapInWithIndex      = "swapInWithIndex"
)

const (
	NearVerifyRange  = "get_verifiable_header_range"
	NearHeaderHeight = "get_header_height"
)

const (
	EpochOfMap          = 50000
	EpochOfBsc          = 200
	HeaderCountOfBsc    = 12
	HeaderCountOfMatic  = 16
	HeaderCountOfPlaton = 430
	EpochOfKlaytn       = 3600
	HeaderOneCount      = 1
	HeaderCountOfBttc   = 64
)

var (
	Big0 = big.NewInt(0)
)

var (
	HashOfDepositIn = common.HexToHash("0xb7100086a8e13ebae772a0f09b07046e389a6b036406d22b86f2d2e5b860a8d9")
	HashOfSwapIn    = common.HexToHash("0xca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5")
	HashOfMessageIn = common.HexToHash("0xf4397fd41454e34a9a4015d05a670124ecd71fe7f1d05578a62f8009b1a57f8a")
	NearOfDepositIn = "150bd848adaf4e3e699dcac82d75f111c078ce893375373593cc1b9208998377"
	NearOfSwapIn    = "ca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5"
)

var (
	Mcs, _         = abi.JSON(strings.NewReader(McsAbi))
	Bsc, _         = abi.JSON(strings.NewReader(BscAbiJson))
	Conflux, _     = abi.JSON(strings.NewReader(ConfluxAbiJson))
	Klaytn, _      = abi.JSON(strings.NewReader(KlaytnAbiJson))
	Near, _        = abi.JSON(strings.NewReader(NearAbiJson))
	LightManger, _ = abi.JSON(strings.NewReader(LightMangerAbi))
	Map2Other, _   = abi.JSON(strings.NewReader(Map2OtherAbi))
	Height, _      = abi.JSON(strings.NewReader(HeightAbiJson))
	Verify, _      = abi.JSON(strings.NewReader(VerifiableHeaderRangeAbiJson))
	Matic, _       = abi.JSON(strings.NewReader(MaticAbiJson))
	Eth2, _        = abi.JSON(strings.NewReader(Eth2AbiJson))
	Platon, _      = abi.JSON(strings.NewReader(PlatonAbiJson))
	Other, _       = abi.JSON(strings.NewReader(otherAbi))
	Bttc, _        = abi.JSON(strings.NewReader(bttcAbi))
	OracleAbi, _   = abi.JSON(strings.NewReader(OracleAbiJson))
	ProofAbi, _    = abi.JSON(strings.NewReader(ProofAbiJson))
)

type Role string

var (
	RoleOfMaintainer Role = "maintainer"
	RoleOfMessenger  Role = "messenger"
	RoleOfOracle     Role = "oracle"
)

var (
	OnlineChaId = map[msg.ChainId]string{}
)

var (
	ConfirmsOfMatic             = big.NewInt(10)
	HeaderLenOfBttc       int64 = 10
	HeaderLengthOfEth2          = 20
	HeaderLengthOfConflux       = 20
)
