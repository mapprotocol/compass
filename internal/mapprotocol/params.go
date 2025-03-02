package mapprotocol

import (
	"github.com/mapprotocol/compass/pkg/msg"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MethodVerifyProofData        = "verifyProofData"
	MethodUpdateBlockHeader      = "updateBlockHeader"
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
	MethodOfMessageIn            = "messageIn"
	MethodOfMulSignInfo          = "multisigInfo"
	MethodOfProposalInfo         = "proposalInfo"
	MethodOfSolidityPack         = "soliditypack"
	MethodOfNodeType             = "nodeType"
	MethodOfMptPack              = "mptPack"
	MethodOfSolEventEncode       = "solEventEncode"
	MethodOfSolPackReceipt       = "solPackReceipt"
)

const (
	NearHeaderHeight = "get_header_height"
)

const (
	EpochOfMap         = 50000
	EpochOfBsc         = 200
	HeaderCountOfBsc   = 12
	HeaderCountOfMatic = 16
	EpochOfKlaytn      = 3600
	HeaderOneCount     = 1
)

var (
	Big0 = big.NewInt(0)
)

var (
	TopicOfClientNotify      = common.HexToHash("0x7063ee7ac21ca792eb7d62d3a65598a5c986c4b0f7bd701aa453eb8a1387c956")
	TopicOfManagerNotifySend = common.HexToHash("0x6644f11ec136e82ae3a252660a2fea9e5d412868cd38474ba2ba564b8f19cb73")
	NearOfDepositIn          = "150bd848adaf4e3e699dcac82d75f111c078ce893375373593cc1b9208998377"
	NearOfSwapIn             = "ca1cf8cebf88499429cca8f87cbca15ab8dafd06702259a5344ddce89ef3f3a5"
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
	Matic, _       = abi.JSON(strings.NewReader(MaticAbiJson))
	Eth2, _        = abi.JSON(strings.NewReader(Eth2AbiJson))
	Other, _       = abi.JSON(strings.NewReader(OtherAbi))
	OracleAbi, _   = abi.JSON(strings.NewReader(OracleAbiJson))
	ProofAbi, _    = abi.JSON(strings.NewReader(ProofAbiJson))
	TronAbi, _     = abi.JSON(strings.NewReader(TronAbiJson))
	SignerAbi, _   = abi.JSON(strings.NewReader(SignerJson))
	PackAbi, _     = abi.JSON(strings.NewReader(PackJson))
	GetAbi, _      = abi.JSON(strings.NewReader(GetJson))
	SolAbi, _      = abi.JSON(strings.NewReader(SolJson))
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
	ConfirmsOfMatic       = big.NewInt(10)
	HeaderLengthOfEth2    = 20
	HeaderLengthOfConflux = 20
)
