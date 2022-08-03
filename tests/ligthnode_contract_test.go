package tests

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/mapprotocol/compass/internal/near"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"

	"github.com/ethereum/go-ethereum"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	maptypes "github.com/mapprotocol/atlas/core/types"
	"github.com/mapprotocol/compass/mapprotocol"
	"github.com/mapprotocol/compass/pkg/ethclient"
	utils "github.com/mapprotocol/compass/shared/ethereum"
)

func Test_NearMcs(t *testing.T) {
	bytes, err := ioutil.ReadFile("./json.txt")
	if err != nil {
		t.Fatalf("readFile failed err is %v", err)
	}
	data := mapprotocol.StreamerMessage{}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		t.Fatalf("unMarshal failed, err %v", err)
	}
	target := make([]mapprotocol.IndexerExecutionOutcomeWithReceipt, 0)
	for _, shard := range data.Shards {
		for _, outcome := range shard.ReceiptExecutionOutcomes {
			if outcome.ExecutionOutcome.Outcome.ExecutorID != "mcs.pandarr.testnet" { // 合约地址
				continue
			}
			if len(outcome.ExecutionOutcome.Outcome.Logs) == 0 {
				continue
			}
			for _, ls := range outcome.ExecutionOutcome.Outcome.Logs {
				splits := strings.Split(ls, ":")
				if len(splits) != 2 {
					continue
				}
				if !ExistInSlice(splits[0], mapprotocol.NearEventType) {
					continue
				}
				t.Logf("log is %v", ls)
				//if !strings.HasPrefix(ls, mapprotocol.HashOfTransferOut) && !strings.HasPrefix(ls, mapprotocol.HashOfDepositOut) {
				//	continue
				//}
			}

			target = append(target, outcome)
		}
		//fmt.Println()
	}

	if len(target) == 0 {
		t.Logf("data is %+v", data)
		return
	}

	cli, err := nearclient.NewClient("https://archival-rpc.testnet.near.org")
	if err != nil {
		t.Fatalf("unMarshal failed, err %v", err)
	}
	for _, tg := range target {
		t.Logf("tg %+v", tg)
		blk, err := cli.NextLightClientBlock(context.Background(), tg.ExecutionOutcome.BlockHash)
		if err != nil {
			t.Errorf("NextLightClientBlock failed, err %v", err)
		}

		clientHead, err := cli.BlockDetails(context.Background(), block.BlockID(blk.InnerLite.Height))
		if err != nil {
			t.Errorf("BlockDetails failed, err %v", err)
		}

		fmt.Printf("clientHead hash is %v \n", clientHead.Header.Hash)

		proof, err := cli.LightClientProof(context.Background(), nearclient.Receipt{
			ReceiptID:       tg.ExecutionOutcome.ID,
			ReceiverID:      tg.Receipt.ReceiverID,
			LightClientHead: clientHead.Header.Hash,
		})
		if err != nil {
			t.Errorf("LightClientProof failed, err %v", err)
		}

		d, _ := json.Marshal(blk)
		p, _ := json.Marshal(proof)
		t.Logf("block %+v, \n proof %+v \n", string(d), string(p))

		blkBytes := near.Borshify(blk)
		t.Logf("blockBytes, 0x%v", common.Bytes2Hex(blkBytes))
		proofBytes, err := near.BorshifyOutcomeProof(proof)
		if err != nil {
			t.Fatalf("borshifyOutcomeProof failed, err is %v", proofBytes)
		}
		t.Logf("proofBytes, 0x%v", common.Bytes2Hex(proofBytes))

		all, err := mapprotocol.NearGetBytes.Methods["getBytes"].Inputs.Pack(blkBytes, proofBytes)
		if err != nil {
			t.Fatalf("getBytes failed, err is %v", err)
		}

		//input, err := mapprotocol.NearVerify.Methods[mapprotocol.MethodVerifyProofData].Inputs.Pack(all)
		//if err != nil {
		//	t.Fatalf("getBytes failed, err is %v", err)
		//}

		fmt.Println("请求参数 ---------- ", "0x"+common.Bytes2Hex(all))
		err = call(all, mapprotocol.NearVerify, mapprotocol.MethodVerifyProofData)
		if err != nil {
			t.Fatalf("call failed, err is %v", err)
		}
	}
	//t.Logf("data is %+v", data)
}

func call(input []byte, useAbi abi.ABI, method string) error {
	to := common.HexToAddress("0xa44b62879B9FfE422615CBccB993E090D93fD0eE")
	count := 0
	for {
		count++
		if count == 5 {
			return errors.New("retry is full")
		}
		outPut, err := dialMapConn().CallContract(context.Background(),
			ethereum.CallMsg{
				From: common.HexToAddress("0xE0DC8D7f134d0A79019BEF9C2fd4b2013a64fCD6"),
				To:   &to,
				Data: common.Hex2Bytes("0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000001ba00000000000000000000000000000000000000000000000000000000000001b341ceeb2948695686ac57413ccafc90b61be244a5d320453caa40777e67eaaae0c19655914c86063bba5ab24a58483ec9d5317e9f12900d6a6064fb86fdbcc54cece4ebb0500000000d1fdc2ed7b6c50034c713bc33ccb9844c9d44669982a624446e26450947460d4764a938de977675d696c8cc5c9b4beee59fc8b2f72da54cc09ba39527a998caebebef7cd788da74c0865120b5f80bde93cd1ee94eb8e26e7119a9854c717a4da3017c4f0e804e2547874ae3910948ba0fedf535e7db1edaed36ef860e6a242cc4227aea3a7590617bafa0cdc0d56db3c4ae1787cd42a8a8e2bc7647823d94fb4d08a110e116728e35cada489c55d48eb24dc87abf3bd30e1d6e114d9cb279418330e683c8b6a2b4714fb7d9dca3983a37f7fddf4811d29ad6562d3352feff4ea65b44f97ec06e5f9013800000000170000006c6567656e64732e706f6f6c2e663836333937332e6d30009012890b00163561ec8bd99da8715c61c32b5be92f3dc00acd7cd9479f319298f322f5cf632f43259797336fbc02000000050000006e6f646531000e8202a3ff22a3df1bf2221f5079005e0bc1438437a2811d513e795d797a58249c764745414d32a0db3caa317201000000050000006e6f646530000e8202a3ff22a3df1bf2221f5079005e0bc1438437a2811d513e795d797a5824dd9c5a6f4ad224e3cb8c28ca1001000000050000006e6f646532000e8202a3ff22a3df1bf2221f5079005e0bc1438437a2811d513e795d797a58240a5da92403a055012a8396811001000000050000006e6f646533000e8202a3ff22a3df1bf2221f5079005e0bc1438437a2811d513e795d797a58245e0db9440def960d199d924a8a00000000160000007374616b65642e706f6f6c2e663836333937332e6d3000b2b3e259cc5bfbe39913e02b6430d29cda2ea3487173f15589b0374186710797ef17ec19c1e2c88edc192d543a000000001c0000006d61737465726e6f646532342e706f6f6c2e663836333937332e6d30007a34a8274dab2f0afa843f39c3f259a2db3e603aad91c3ecb6a079fce6052c99c583e47249cfaca2a2f9951b37000000001600000030316e6f64652e706f6f6c2e663836333937332e6d30002850e20b8ce610aee3d913441ff7ade411d5524f445dfd8c2fc7784d0e5155ad6d413184c33b305f4fea0c453100000000130000007032702e706f6f6c2e663836333937332e6d3000373dca6a91c07ad49f86a8cac22b591eccec525bbe91ab11515b796a335ea67fa349c9a785a11b478ab7dea92f00000000170000006e6f64656173792e706f6f6c2e663836333937332e6d30000ff05f1e15d447bfd0fc36be2416b585b3c4238cdfc644d3da84ef09e1ea2d4a83f2a80227a6e1d64030bbaa18000000001a00000074726962652d706f6f6c2e706f6f6c2e663836333937332e6d3000a9b2d3c444791eee5f61f9907a0fc9ba6cffecf1ef27cbec9b33fdadcabcfb2370c89c5193b7555fc014634715000000001900000063686f7275736f6e652e706f6f6c2e663836333937332e6d30002491a39951692ea9824a9821c6ffadb15aed6801ebbb587d225d4bb404110dec0b5c4e5088494b9f9a68e64114000000001a000000666f756e6472797573612e706f6f6c2e663836333937332e6d3000885ad5cbf24be2f85e01da691f16a0beb7d012fdeeb1a1d2f7efa8a14a48daba1b01aea9e938afb18f4bebb013000000001a00000063686f7275732d6f6e652e706f6f6c2e663836333937332e6d30004f395f192fe94ea65559accb20bcf144700d3554a60553de67f05480583977598f8757661988767d2dd8c8941100000000120000006e692e706f6f6c2e663836333937332e6d3000e8a88ee26c0c29b07da2ed2bea825cf08acbb57ee98f534aa7473eb27faa5d7588eeaf15ec35c454a2bf634911000000001b00000063727970746f676172696b2e706f6f6c2e663836333937332e6d3000de6c9696f96a93e27bdb465b90eaf00718fae027c339202c379cd18b9f354a182269228b9bb3cc652a4ff3500d000000001f00000070617468726f636b6e6574776f726b2e706f6f6c2e663836333937332e6d3000a7891814823af0fce30a3e5037534803fed58c0ab7e0de2aea9e8ef5982c3e31c1e940b644c51e2adc3ca4070c000000001a0000007374616b656c795f76322e706f6f6c2e663836333937332e6d30005bdc205a212c4573d32375b2f93ad215c21a34508ef89123a3ac5f8492552fd7154e513f9784012710806cfa0b00000000160000006175726f72612e706f6f6c2e663836333937332e6d30007fdc7a529b631deb0c6c75e1893696781b3ff4d3e4a8fadee70dce85b80113c80d19eca08d50cb26b2c30b4a0b00000000190000006672657368746573742e706f6f6c2e663836333937332e6d3000448cb66c22d02b68c80a48d87b34b3d4b86a0e9f9d3d6c6415cddedfbecf0b15fb28d3c890390893d99123220b000000001a000000736f6c696473746174652e706f6f6c2e663836333937332e6d3000b903b5737b3809a2ebe79ecd6da5965b0db163c0858b6a0fc92be9e0cb81b3d2ad1a00abad885da56ee1ab090b00000000190000006e616d646f6b6d61692e706f6f6c2e663836333937332e6d3000844164539a671bfe030f8215f07245da1d1f067c7bd7cdd88352cf2a7fc99687682df85d12e9ce17cd9b49f30a000000001900000073686172646c6162732e706f6f6c2e663836333937332e6d3000c095b93a6e418aa2fddaff90cff363d87c6ebcb08731ac012218a6a543d0790f07f53ee7fa7768da538ac17b0a000000001a000000626c6f636b73636f70652e706f6f6c2e663836333937332e6d30004eed9f8ec11536f0c3c8f5171365b2524afb37d07a2052e433654a2c514e3e39678f727cf6f7590ad27b0f620a00000000180000006c6561646e6f64652e706f6f6c2e663836333937332e6d3000acc27128aea3731cad4f045b0d011892f85857867a89305393b9bce796a316d572fea1d8596f596bcae756540a000000001b0000007374616b657373746f6e652e706f6f6c2e663836333937332e6d300026366f84610fdd95923c16f9575e609ac5537adc7d5beb7b472ee7fea4db03791eea24976357cc82d83f1a4d0a0000000015000000616c3363352e706f6f6c2e663836333937332e6d3000a08174cb2460a626fbcca253e279ad175f02e12b7b358f07c10aacbb921b8e7ac69adf28ca52788f2da3cd360a00000000270000006f7074696d757376616c696461746f726e6574776f726b2e706f6f6c2e663836333937332e6d300098a1abd9658a7a794fb6328ba64b1eb62389822407328aa06fa3ad6d729c277092b07968c2b0fcd5837c2c280a000000001800000067726173736574732e706f6f6c2e663836333937332e6d3000242271a13684d6ac3672aed39c96586d6376c5e2cbaa0b45e56422b15f230cfe5d693ed40b6c2acf97d7cd1f0a000000001e000000626173696c69736b2d7374616b652e706f6f6c2e663836333937332e6d3000a73adabf811046d06cb097d6d6d967af363bf9485b8f3c63f968522ea2ff7b752ed526c9f7881fcf5874d7ea09000000001600000073687572696b2e706f6f6c2e663836333937332e6d300085872e16b7806fe2b385d4fc65609a68caade203edbf2f147209a1052ac3676bc1d9e92c050c46a4e421b0c7090000000018000000647372766c6162732e706f6f6c2e663836333937332e6d30004a75120cf200659da75ee453d629fc0a411e5764652e441a3119d67097e20a985e9f206fc14bfba77f5b7fa80900000000220000006368656c6f76656b5f697a5f6e61726f64612e706f6f6c2e663836333937332e6d30006a3459bb11fb149f903ec4b26ffa6df2867cc667868d44833c5a185b3c2ddd863b444828cb5969528d5e6c8409000000001b00000070726f6a65637474656e742e706f6f6c2e663836333937332e6d30001c580cc69128eeecb4a7de65d59411596bf3b15ab7cda621521353440e20e307b688ba06880293d9f0ce656a0900000000150000007a657473692e706f6f6c2e663836333937332e6d300056fc051c663cdef9b8fae2d29a6b3258e7e18d5178dc37f18280da2c6b773c2291823a2c8fc12a526fbbda2709000000001c0000006c6176656e646572666976652e706f6f6c2e663836333937332e6d300094908929e2b7913962d2143fe91e511e72601553d0549630503403ec03271f11079e53c90069543db277d50b090000000016000000746179616e672e706f6f6c2e663836333937332e6d3000e10e726a5533ee7ef579195cbfee120cb04c9473f3c24978429ea0f066f7199705e4fa81d017f8ade3311700090000000019000000657665727374616b652e706f6f6c2e663836333937332e6d3000317f139e9df2e803cee46652ac7701aec09154c3f8cfd13acb24180069066fa2cb8183c727afcc29ea570de508000000001c000000696e66696e6974656c6f6f702e706f6f6c2e663836333937332e6d300018bf01eeff2dd463487016ee068143b22832530485783a7449f92769ab2fd422aaf63ae626ccd3f14499335508000000001600000062666c616d652e706f6f6c2e663836333937332e6d30003a08b4bf81201c71cfd0ec03b7e32e64367a20921b1d2b3153bc0c377b65809e6dc7855b402cbd5db0fdc82508000000001f000000726f7373692d76616c696461746f722e706f6f6c2e663836333937332e6d3000187262f21b61daa24807e2ea1a68369ee7152da684c8c23f22216dfc5b9e5a2a6cf705223e1d231b60164ce606000000000e00000067322e706f6f6c2e6465766e6574009c8e2cda94e7fd459f1d2e61bf6a92b5ace41183c0fe3aa9a298aa657951205980c9d2691c8cfe0df6303a4a0200000000140000006b696c6e2e706f6f6c2e663836333937332e6d3000a0e96a90b0e21954c40e5e2fc7d56c412f1d52af0a8ea453d088a9c3659b8abd768904897748a8ad1cdd09a70100000000130000006962622e706f6f6c2e663836333937332e6d300063606867b3a0eea0422e8001fcd11c3c59c7e287c7784f5e57895549af5a29a50edd96d6b4affd45e12bb9a7000000000028000000776f6c66656467652d6361706974616c2d746573746e65742e706f6f6c2e663836333937332e6d3000a96404075c2deb780d42f0b46f6d1001a3b5c52677a12eb3977cf1c41174e03d81bbd749c5d7a3b9984d2a960000000000190000007477696e74657374312e706f6f6c2e663836333937332e6d30005c4e18870c8c092d075511c19864ab455ec1d13ee17ad70adff220817dfa51bec5b97e1c1b5c7a693802e789000000000019000000626565317374616b652e706f6f6c2e663836333937332e6d300075e6c94cb21a7344e227107ca2f14ceac03be64a461c6a0e711be7128395ddfe63c4c7599c77b975c495ec7f0000000000180000006a7374616b696e672e706f6f6c2e663836333937332e6d300009f7aaacc1dedaf1ccbe4402ed3d78458f3e39ccce3993c561d85f1312a718e3f36a45c41775180da245fd77000000000019000000636f646573746174652e706f6f6c2e663836333937332e6d3000293e0bfb9d5e21ca8d3261eb54b9deeab98e48c82b9be6f74399613ce83a340f6a73d4ea7e53b890fe38fd770000000000180000006c6173746e6f64652e706f6f6c2e663836333937332e6d3000680294de4a16790d40c41de999602a7cf9f2a9383fd793a42defd1788cba8a9872311c1f7705bec86b042f770000000000290000006f6d6e697374616b655f76352e666163746f727930312e6c6974746c656661726d2e746573746e657400c34054d4e883918f2ba6495f84d02ce2d604bc91ef40e627ecce9cf145f315d6475ddd6c64a8e20363b07a7600000000001b00000067657474696e676e6561722e706f6f6c2e663836333937332e6d30004193e2cd0630b71f1af526515f63863aad964e08eb451d2dd9454ec6041a4e7d47a4a30b018fe8a271c1d17200000000001b00000063727970746f6c696f6e732e706f6f6c2e663836333937332e6d300093630e39084f59480edce26296f4c225e729ba327b91741cd0a9055c7a993f4100faf3c67b8acf521f7193710000000000210000007374616b696e67666163696c69746965732e706f6f6c2e663836333937332e6d3000240fa5e7442f912ecb8c0bb2d0e83e0b3a4ca69966a09dfc5940806fd848eeed2dda7b3a825621eb49b41f710000000000210000006b74312e6e62687374616b696e676661726d666163746f72792e746573746e657400fc6953d6b90f1ef6db009f43f9aea925eacbf831ec6a266ae2a25e56b8fa9335131239fba1386d125bab116b0000000000170000006b757574616d6f2e706f6f6c2e663836333937332e6d30006eb206c6ec9b23d923f142dc8eeca627b2f6fcae9f7d1a21a4965eb7592506e5392312227d098800f98d9a6700000000390000000100ee9319e8d03c4af7500fbfe3b266d531489f577311c490b23b60886cc364b4d2e12584b2e22678b47331d253c531eb2b395eef5f554aa0e4e4d8b9fba37cd6050100c0a6af229387e5fa1e4e9c2d14fe13661c4af7d6e3cce5918ff6962dd1117160871fe995f9f4117ed53b6b90c6eedb1c26425cc67c2f0e4afcd99ab5d0678501000100c0a6af229387e5fa1e4e9c2d14fe13661c4af7d6e3cce5918ff6962dd1117160871fe995f9f4117ed53b6b90c6eedb1c26425cc67c2f0e4afcd99ab5d06785010100c0a6af229387e5fa1e4e9c2d14fe13661c4af7d6e3cce5918ff6962dd1117160871fe995f9f4117ed53b6b90c6eedb1c26425cc67c2f0e4afcd99ab5d067850100010020876d4ae3ee94b32394f6ad785e1b2d3e83b8e7a033516bd100b4e19e16421da41c6fe499c01253ab999c6f33e51ee162dd9cc298b1c5e12ee3020b87ccec0701009a0d2f51c994071d73a1944ef3ff6c7737ef090194e7b5fef796df6054191c57c9c1d9e4baea18e6f0aefdf247fe06434558b0ce8164f3f35b8b62d88eeb6f0f01000733e4ff71d887385254f7286f1368d8d401f2de70bd8e0566c97bbd5be207d5d6e4a4d6b996d35376f3e57c27e1ef8adebd5ea35a2c9e3df4155d6495f8930b0100cf18dbf0d4847c583d22a9344f6c7d823ebb2e1b3dbd2251bf61293919a74318d60fb7fb31026a1dd063b37eea26e64e320152a563fd65c8ee873c05e3faf9030100ce46f46e85b401a813a064ea57dcea1831e69aaccc14f76b3409eeef7ca0f72c1c40071565f879fa8af7eacbc6da55149a340adc744c96e43c0742e09f0ff60f0100b7ea8b7c4eaa0cd5c8a82f197e1505711d6a5eb27b3c4ebf0e75b59bcafe5a6786b4da232dc15da4446bb5f8db755cc2e46bdecf53d07804d4a62f8f105292070100d2de93ecbe840c71a111d1266056c3633d9d11676783a216196c0a07da1ad3c7b80e7437ead3c856f1992bc1b12fba5986c277cc7e74605afcc5df44a3987a0a01007bf1f8dcdc64efe89e319d4f92c3d61b6ccd6d44e4cfe24ac7e03a09c5b17b2c11d1dc948694caa2fdcf6c1f811f8caa45be26f588a0b60a2da7d3bd9d94c40f0100232785baca773dac41709769094c4d17e2434efb8f809c3a05cf96aaa461a6afbe858905715533a6927e290f423d550d2db164f4e98acfb3ab71349816b8c0040100965ced3dae6d2f38a5c054318290da44a64e00691efe7d1ce0103688b0f133ef999cc2572f34a9bfbde49b38dd329baaa482d64a78de606ac82a20d7db1823030100d42f75b6e0c81424a462b8f9f8dc57a7a9188896ebc49a0c0bb26122e562249aa2c89e17c3191c7e3ec33b552eddfeaeaa41d01270b8ab25cf3359b70275060201000ff6699f73d9bab3aaba840ce56c6b66d97607abd635a21815ffe390bfc75ed9668be5be1c5648678a5bd21c55612bc7197876a66343fc134df4e90cd0c5e00c000100773d317d45fb294f877240e1763f2e205422668db5b38c094b95e22fd0aab2ae62b1faf5410c82bc716e33c0de74bac980f63ab76c074c56b9297aa2855b180700000100b9d179f84094d150f881600534a38cc1d01090f62c66167c9c67a511b9907ee3dbdd1bb1b05c2cf5de9489a4c03826bb463f5f63e148363ce83cbd841aea15040100372231ace5a821b93bf21ebbe5dffaee4c87e64f7bc0083da704e0ed0880277772c79aef0a0847e13971ac1c4f957ff9c8096f846eb319f6e3c83cf46d8a960f0100d63c451e4b23cc702e4a87a43dee12068b679a68261e5f77f88c4296ce6772b19f95b66fe40e10a0f4f85263a6f6cc05f9a558164ff740b29cf6a4e2095466090001000543a705c1b5a7f395b51c407e149f07dd9f15b999c850285c8cd900d4a3035a6f4919e8b0a5ac95f1af652c78ebc9d9f6d0c58d656410b011e8f0309e876707010089eb87df7a29a564a1071557fc6d4b09d47775e2785cac9748286c5643027863ca3204ee87a35a4784e90e08192c090649a05b21f0f25fa7e535657d23efe20f010020a06ccfaad9f734f844206142b2ac3405e1b7f17bf21df608ad036a16ec9aa084a6cd31570ea07d778b4b0bcb9cc9e7fb724e8f6b26a2c3294abddd1aa8660f00000100121f63db3f85716c75faacabc13a5515a1bd8585fe1647f6e3d58cda2ddb99383cd35209809b41aa95b79660c4a40bf6b21e6e56c19f5363dd690a2639775a08000001007d77f722b2c3c60edb14478b69dfff4c74e17705087693c5eaceddfdb2a9710d9a1e959cd3167755b57e6bcdf7f0a08edbb51e77de4f8c45bced32e91dfe610101004cdf97ac35b593108308bda97e9903df786d760b58850c7f0ecf6941460cf3a60407708637e18e6d2169d3270522dc3ad589341056b7e28ad6393833c592580a000000000100c8f4dbc21fb90f5f101eaa18f99383da33af1c7026a1438a2384f233b54567c0bbe5d705e16fea7d0e8cdc7ac807d4d3821f7601e24c6a4d6ec633cf23b968000100d82a90371bae50b51e1a40652a1038d308df1dbb88540efe24ea3d012ae194ddd7c006d2bc72f50f9744c505def973b811ff62b23552419075c1021f009d5908010081546f50cdd1eff7bba9c4933916fc280fc31beeb9326cd19a1bb74702f9ad0e2375682a8361aabe1ea34c9fc7be06d817f1f788e0156ee4263afbd8e9541308000001001020810f57360e31f9548d9edbff3dc8bd1b2d065408011b2b169bc0b5e3159aa152d2e3ca71014c51abc496de3bfe19b7a44628207a761d3f3038c8313ced0b00000100e634f8c2a09d82c602d3f8b3348272a918a634077910a63c315177fb84df9bbaf6dc4d5bd9ec50ff6c7c5e79e9cc7058b7c7b2200eb54b6217fcdaa887e29a0a0001002630a0a1e9b10c8925f432a50999bfc9bcb949641ac86436baba544185b6cf945592bd630e4058b843765392d272d6d7a19e20fa8983562f76a40f00536003010001006a74fd80978b4dd481d92b8670e1a51163e3eb129c04d6c882a72900a0dab81379a37c7106ad74c5d23b5a25461fd91366fa2c7dc355064324ac825ccfc7860900010097631cb60afce35f4562f4ba899d002252170aafbd92039d1a6c75018b20d4a10618b342acfb514567cb0714ddda82c5c59d24f3a9f333a31e42df33b57a6c0a0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000067d03000000e304764586d4788096bbf04b83313b07b87ba4d6b2a8fa3d2cb83ed69e09c41500889a76a78852aa39aecba0f5895d5fe2466610e7da81fa0e620b81290b51833001dc88b9b6dbedc9cefb73ee18a2e71be1811fb5713848fdc3b49431adff27c8300015c1e43670a338ede45ee6d60123072ea29f1ca4de07bbe2c5c891ca4ed36289c78acf4030563f984fda294f5344eee06941114edc32355befa22790cad07b3201000000640100007472616e73666572206f75743a2031323030303030303636373432653730363136653634363137323732326537343635373337343665363537343066303030303030373036313665363436313732373232653734363537333734366536353734363531333565326266346634303264393163313435303736626261366335333262323734633530303139326430353539646462333837316239376530393664653533343134353465303030303030303030303030303030303030303030303030303130303030303030303030303030303030303030303030303030303030303032613030303030303330373833373336333033373633333936333634363433373333333336343338363336343631333036313336333433343338333333393635363333323632363136333335363636313331333833303635363433343634303030303030303030303030303030303030303030303030303030303030303030303030303001000000c45467ee76d18678a978a1f86a3c9af0c67eefecd11883dadb6827f95817d180f4425ab09a03000000745c2cb07ea77b1500000000000000130000006d63732e70616e646172722e746573746e657402030000002230220200000083ec86f23821cdae77348058febce00024fde89ab9ad85947dc3d034966ef06a00bc5c24c31fd2c7dd8f33748dff20ebd142932d8f675892b5cc768e3c6afb825b00b7ee7f391a1e90050ff83b02e19227f9fbdbf2e61e618eff7112582985b67ef0eeb365f56cb2244f30e2cd4ed48d0db15f597e076ee75b65dccaeb0b9d0aa256658eba05000000004e1fad9c75f44a34b178c178c2f9285692fe1a5e67ac66c96dbe58c11a107da4d1fdc2ed7b6c50034c713bc33ccb9844c9d44669982a624446e26450947460d42e4ec20fa6dc7f29c93c9eadcd991a405761cd825671e800eff231f648b099a89a30ca125b897902ab57ebcfdf1462760a9e1d857a8f2c06fd184b180b1bfd5bd672c116ba2c061713746cb7fce03e631de1877f579b27a3131e41d0b58d126f544a02142257490cae979162fa40996d83dc709c41474d6711732c2b79d40300807e818e167fe60115000000b7ee7f391a1e90050ff83b02e19227f9fbdbf2e61e618eff7112582985b67ef000232a2041ac97475912a20bd1eb822caef6fa8d4d79996068ded7da1e7d65ab2b00a1cae7a265839fab912ea28918feef0b55a86c54231e3f2249650f999f6a804001bd2176efd3d7e0b6408e105a355ce96b5327bf160ea0482f077bc954c788a7bb014e7f39ebaf8f18e8c8356bd7bb46b4b6f95f5ac6b27232558cebfa29f9e0328a002dd871cde1782e06cb284de82836400fdfa6e36a1e8530331021675a56a7384601ab512f0ca24ca29669ccfb6e69003f74c5a9233b785c87ac1f8339f90054a76600c9bc66be2d3deed7b2c421ee797ce4812543a3c7d350efd17ad7ab1cfc228f5c017eb27a41ef0b4ef5ecb79caa6cd033a6e837f73cfc9f6d49bf8b45203b0da72d001d0422139982484f76c7c84e33e00333b17cc751bc8c88873bc727942eeb494000caf97614839e34adaebfd203c2f07eae37a1274f3e21a001af3ba090d73f03fc0075a17885780083c7db372d9a0b591c7e43bbe67464e8eb2cef5885894dedde330033b5857c4f010b661793e91df6d465543612dffe57e55689d704d47e03297fb10114ef85330ee306317fd101ab3eefcc23e715593939263732260c81f905def7e50119e02532bdb262f9a74064077795849ca590d0b30f6b3c75e8ceb3ac7bc7a37700fc7d0ec4d372c5ff986cbc397f1271129252e9e0bec16a12e8b62bdb335a0f2800a1a1154d7752623ffd035d064174e07acf164ec808319fe6ea897379514073bf01dec846f9f6a7e665447461105ea256a0a2ec3f9240f36fb57c1d8cb6e495a95c00bb83e2f3e6b69ec0f71575cd0479f5f72527983c22798fd0c7d9927a7f11a11c009fc14f87e6739602ad0e5f1211fcb9e0d59826b0d35d326d468842b73aacf66a005a8ca7fe98ead701132822284b29306097b3506253c07911a1450eca69ad727100000000"),
			},
			nil,
		)
		if err != nil {
			log.Printf("callContract failed, err is %v \n", err)
			if strings.Index(err.Error(), "read: connection reset by peer") != -1 {
				log.Printf("err is : %s, will retry \n", err)
				time.Sleep(time.Second * 5)
				continue
			}
			return err
		}

		resp, err := useAbi.Methods[method].Outputs.Unpack(outPut)
		if err != nil {
			return err
		}

		ret := struct {
			Success bool
			Message string
			Logs    []byte
		}{}

		err = useAbi.Methods[method].Outputs.Copy(&ret, resp)
		if err != nil {
			return err
		}
		if !ret.Success {
			return fmt.Errorf("verify proof failed, message is (%s)", ret.Message)
		}
		if ret.Success == true {
			log.Println("mcs verify log success", "success", ret.Success)
			log.Println("mcs verify log success", "logs", "0x"+common.Bytes2Hex(ret.Logs))
			return nil
		}
	}
}

func ExistInSlice(target string, dst []string) bool {
	for _, d := range dst {
		if target == d {
			return true
		}
	}

	return false
}

var ContractAddr = common.HexToAddress("0xA7D3A66013DE32f0a44C92E337Af22C4344a2d62")

func dialConn() *ethclient.Client {
	conn, err := ethclient.Dial("https://ropsten.infura.io/v3/8cce6b470ad44fb5a3621aa34243647f")
	if err != nil {
		log.Fatalf("Failed to connect to the atlas: %v", err)
	}
	return conn
}

func dialMapConn() *ethclient.Client {
	conn, err := ethclient.Dial("http://18.142.54.137:7445")
	if err != nil {
		log.Fatalf("Failed to connect to the atlas: %v", err)
	}
	return conn
}

func TestLoadPrivate(t *testing.T) {
	path := "/Users/t/data/atlas-1/keystore/UTC--2022-06-07T04-22-55.836701000Z--f9803e9021e56e68662351fe43773934c4a276b8"
	password := ""
	addr, private := LoadPrivate(path, password)
	fmt.Println("============================== addr: ", addr)
	fmt.Printf("============================== private key: %x\n", crypto.FromECDSA(private))
}
func TestUpdateHeader(t *testing.T) {
	cli := dialConn()
	for i := 1; i < 21; i++ {
		number := int64(i * 1000)
		fmt.Println("============================== number: ", number)
		header, err := cli.MAPHeaderByNumber(context.Background(), big.NewInt(number))
		if err != nil {
			t.Fatalf(err.Error())
		}

		h := mapprotocol.ConvertHeader(header)
		aggPK, err := mapprotocol.GetAggPK(cli, header.Number, header.Extra)
		if err != nil {
			t.Fatalf(err.Error())
		}

		//printHeader(header)
		//printAggPK(aggPK)
		//_ = h

		input, err := mapprotocol.PackLightNodeInput(mapprotocol.MethodUpdateBlockHeader, h, aggPK)
		if err != nil {
			t.Fatalf(err.Error())
		}

		path := "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-15T07-51-25.301943000Z--e0dc8d7f134d0a79019bef9c2fd4b2013a64fcd6"
		password := "1234"
		from, private := LoadPrivate(path, password)
		if err := SendContractTransaction(cli, from, ContractAddr, nil, private, input); err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func TestVerifyProofData(t *testing.T) {
	var (
		number = big.NewInt(106020)
		//number       = big.NewInt(4108)
		//number       = big.NewInt(55342)
		txIndex uint = 0
	)
	cli := dialMapConn()

	header, err := cli.MAPHeaderByNumber(context.Background(), number)
	if err != nil {
		t.Fatalf(err.Error())
	}

	txsHash, err := getTransactionsHashByBlockNumber(cli, number)
	if err != nil {
		t.Fatalf(err.Error())
	}
	receipts, err := getReceiptsByTxsHash(cli, txsHash)
	if err != nil {
		t.Fatalf(err.Error())
	}
	receipt, err := mapprotocol.GetTxReceipt(receipts[txIndex])

	proof, err := getProof(receipts, txIndex)
	if err != nil {
		t.Fatalf(err.Error())
	}

	aggPK, err := mapprotocol.GetAggPK(cli, header.Number, header.Extra)
	if err != nil {
		t.Fatalf(err.Error())
	}

	var key []byte
	key = rlp.AppendUint64(key[:0], uint64(txIndex))

	//fmt.Println("============================== number: ", number)
	//printHeader(header)
	//printAggPK(aggPK)
	//printReceipt(receipt)
	//fmt.Println("============================== KeyIndex: ", "0x"+common.Bytes2Hex(key))
	//printProof(proof)

	rp := mapprotocol.ReceiptProof{
		Header:   mapprotocol.ConvertHeader(header),
		AggPk:    aggPK,
		Receipt:  receipt,
		KeyIndex: key,
		Proof:    proof,
	}

	input, err := mapprotocol.PackLightNodeInput(mapprotocol.MethodVerifyProofData, rp)
	if err != nil {
		t.Fatalf(err.Error())
	}
	path := "/Users/xm/Desktop/WL/code/atlas/node-1/keystore/UTC--2022-06-15T07-51-25.301943000Z--e0dc8d7f134d0a79019bef9c2fd4b2013a64fcd6"
	password := "1234"
	from, _ := LoadPrivate(path, password)
	output, err := dialConn().CallContract(context.Background(), eth.CallMsg{From: from, To: &ContractAddr, Data: input}, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}

	resp, err := mapprotocol.ABILightNode.Methods[mapprotocol.MethodVerifyProofData].Outputs.Unpack(output)
	if err != nil {
		t.Fatalf(err.Error())
	}

	ret := struct {
		Success bool
		Message string
	}{}
	if err := mapprotocol.ABILightNode.Methods[mapprotocol.MethodVerifyProofData].Outputs.Copy(&ret, resp); err != nil {
		t.Fatalf(err.Error())
	}

	fmt.Printf("============================== success: %v, message: %v\n", ret.Success, ret.Message)
}

func TestGetLog(t *testing.T) {
	//number       = big.NewInt(4108)
	//number       = big.NewInt(55342)
	query := buildQuery(common.HexToAddress("0xf03aDB732FBa8Fca38C00253B1A1aa72CCA026E6"),
		utils.SwapOut, big.NewInt(106020), big.NewInt(106020))

	// querying for logs
	logs, err := dialConn().FilterLogs(context.Background(), query)
	if err != nil {
		t.Fatalf("unable to Filter Logs: %s", err)
	}
	t.Logf("log len is %v", len(logs))
}

// buildQuery constructs a query for the bridgeContract by hashing sig to get the event topic
func buildQuery(contract common.Address, sig utils.EventSig, startBlock *big.Int, endBlock *big.Int) eth.FilterQuery {
	query := eth.FilterQuery{
		FromBlock: startBlock,
		ToBlock:   endBlock,
		Addresses: []common.Address{contract},
		Topics: [][]common.Hash{
			{sig.GetTopic()},
		},
	}
	return query
}

func SendContractTransaction(client *ethclient.Client, from, toAddress common.Address, value *big.Int, privateKey *ecdsa.PrivateKey, input []byte) error {
	// Ensure a valid value field and resolve the account nonce
	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		return err
	}
	//fmt.Printf("============================== from: %s, nonce: %d\n", from, nonce)

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	gasLimit := uint64(2100000) // in units
	// If the contract surely has code (or code is not needed), estimate the transaction
	msg := eth.CallMsg{From: from, To: &toAddress, GasPrice: gasPrice, Value: value, Data: input}
	gasLimit, err = client.EstimateGas(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("contract exec failed, %s", err.Error())
	}
	if gasLimit < 1 {
		gasLimit = 866328
	}

	// Create the transaction, sign it and schedule it for execution
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, input)

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return err
	}
	//fmt.Println("TX data nonce ", nonce, " transfer value ", value, " gasLimit ", gasLimit, " gasPrice ", gasPrice, " chainID ", chainID)
	signer := types.LatestSignerForChainID(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	txHash := signedTx.Hash()
	count := 0
	for {
		time.Sleep(time.Millisecond * 1000)
		_, isPending, err := client.TransactionByHash(context.Background(), txHash)
		if err != nil {
			return err
		}
		count++
		if !isPending {
			break
		} else {
			log.Println("======================== pending...")
		}
	}
	receipt, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return err
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		logs, _ := json.Marshal(receipt.Logs)
		log.Printf("Transaction Success, number: %v, hash: %v， logs: %v\n", receipt.BlockNumber.Uint64(), receipt.BlockHash, string(logs))
	} else if receipt.Status == types.ReceiptStatusFailed {
		log.Println("Transaction Failed. ", "block number: ", receipt.BlockNumber.Uint64())
		return errors.New("transaction failed")
	}
	return nil
}

func LoadPrivate(path, password string) (common.Address, *ecdsa.PrivateKey) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	key, err := keystore.DecryptKey(bs, password)
	if err != nil || key == nil {
		panic(fmt.Errorf("error decrypting key: %v", err))
	}
	priKey := key.PrivateKey
	addr := crypto.PubkeyToAddress(priKey.PublicKey)

	if priKey == nil {
		panic("load privateKey failed")
	}
	return addr, priKey
}

func printHeader(header *maptypes.Header) {
	type blockHeader struct {
		ParentHash  string
		Coinbase    string
		Root        string
		TxHash      string
		ReceiptHash string
		Bloom       string
		Number      *big.Int
		GasLimit    *big.Int
		GasUsed     *big.Int
		Time        *big.Int
		ExtraData   string
		MixDigest   string
		Nonce       string
		BaseFee     *big.Int
	}
	h := blockHeader{
		ParentHash:  "0x" + common.Bytes2Hex(header.ParentHash[:]),
		Coinbase:    header.Coinbase.String(),
		Root:        "0x" + common.Bytes2Hex(header.Root[:]),
		TxHash:      "0x" + common.Bytes2Hex(header.TxHash[:]),
		ReceiptHash: "0x" + common.Bytes2Hex(header.ReceiptHash[:]),
		Bloom:       "0x" + common.Bytes2Hex(header.Bloom[:]),
		Number:      header.Number,
		GasLimit:    new(big.Int).SetUint64(header.GasLimit),
		GasUsed:     new(big.Int).SetUint64(header.GasUsed),
		Time:        new(big.Int).SetUint64(header.Time),
		ExtraData:   "0x" + common.Bytes2Hex(header.Extra),
		MixDigest:   "0x" + common.Bytes2Hex(header.MixDigest[:]),
		Nonce:       "0x" + common.Bytes2Hex(header.Nonce[:]),
		BaseFee:     header.BaseFee,
	}
	fmt.Printf("============================== header: %+v\n", h)
}

func printAggPK(aggPk *mapprotocol.G2) {
	type G2Str struct {
		xr string
		xi string
		yr string
		yi string
	}
	g2 := G2Str{
		xr: "0x" + common.Bytes2Hex(aggPk.Xr.Bytes()),
		xi: "0x" + common.Bytes2Hex(aggPk.Xi.Bytes()),
		yr: "0x" + common.Bytes2Hex(aggPk.Yr.Bytes()),
		yi: "0x" + common.Bytes2Hex(aggPk.Yi.Bytes()),
	}
	fmt.Printf("============================== aggPk: %+v\n", g2)
}

func printReceipt(r *mapprotocol.TxReceipt) {
	type txLog struct {
		Addr   common.Address
		Topics []string
		Data   string
	}

	type receipt struct {
		ReceiptType       *big.Int
		PostStateOrStatus string
		CumulativeGasUsed *big.Int
		Bloom             string
		Logs              []txLog
	}

	logs := make([]txLog, 0, len(r.Logs))
	for _, lg := range r.Logs {
		topics := make([]string, 0, len(lg.Topics))
		for _, tp := range lg.Topics {
			topics = append(topics, "0x"+common.Bytes2Hex(tp))
		}
		logs = append(logs, txLog{
			Addr:   lg.Addr,
			Topics: topics,
			Data:   "0x" + common.Bytes2Hex(lg.Data),
		})
	}

	rr := receipt{
		ReceiptType:       r.ReceiptType,
		PostStateOrStatus: "0x" + common.Bytes2Hex(r.PostStateOrStatus),
		CumulativeGasUsed: r.CumulativeGasUsed,
		Bloom:             "0x" + common.Bytes2Hex(r.Bloom),
		Logs:              logs,
	}
	fmt.Printf("============================== Receipt: %+v\n", rr)
}

func printProof(proof [][]byte) {
	p := make([]string, 0, len(proof))
	for _, v := range proof {
		p = append(p, "0x"+common.Bytes2Hex(v))
	}
	fmt.Println("============================== proof: ", p)
}

func getProof(receipts []*types.Receipt, txIndex uint) ([][]byte, error) {
	tr, err := trie.New(common.Hash{}, trie.NewDatabase(memorydb.New()))
	if err != nil {
		return nil, err
	}

	tr = utils.DeriveTire(receipts, tr)
	ns := light.NewNodeSet()
	key, err := rlp.EncodeToBytes(txIndex)
	if err != nil {
		return nil, err
	}
	if err = tr.Prove(key, 0, ns); err != nil {
		return nil, err
	}

	proof := make([][]byte, 0, len(ns.NodeList()))
	for _, v := range ns.NodeList() {
		proof = append(proof, v)
	}
	return proof, nil
}

func getTransactionsHashByBlockNumber(conn *ethclient.Client, number *big.Int) ([]common.Hash, error) {
	block, err := conn.MAPBlockByNumber(context.Background(), number)
	if err != nil {
		return nil, err
	}

	txs := make([]common.Hash, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		txs = append(txs, tx.Hash())
	}
	return txs, nil
}

func getReceiptsByTxsHash(conn *ethclient.Client, txsHash []common.Hash) ([]*types.Receipt, error) {
	rs := make([]*types.Receipt, 0, len(txsHash))
	for _, h := range txsHash {
		r, err := conn.TransactionReceipt(context.Background(), h)
		if err != nil {
			return nil, err
		}
		rs = append(rs, r)
	}
	return rs, nil
}
