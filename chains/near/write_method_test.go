package near

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/mapprotocol/compass/keystore"
	"github.com/mapprotocol/near-api-go/pkg/client"
	nearclient "github.com/mapprotocol/near-api-go/pkg/client"
	"github.com/mapprotocol/near-api-go/pkg/client/block"
	"github.com/mapprotocol/near-api-go/pkg/types"
	"github.com/mapprotocol/near-api-go/pkg/types/action"
)

var (
	from     = "zmm.test.near"
	to       = "atlas.test.near"
	endPoint = "http://46.137.199.126:3030"
)

func dailRpc() *nearclient.Client {
	client, err := nearclient.NewClient(endPoint)
	if err != nil {
		panic(err)
	}
	return &client
}

func Test_writer_sendTx(t *testing.T) {
	kp, err := keystore.NearKeyPairFrom("local", "/Users/xm", from)
	if err != nil {
		t.Fatalf("keypair err %v", err)
	}

	ctx := client.ContextWithKeyPair(context.Background(), kp)
	s := `{
		"agg_pk": {
            "xi": "0x08ccd7feb39031341784777bf41ddcbac670e60ee2b466e159e18af9d0b0f565",
            "xr": "0x1e703741166519a7e278c11a8dc96ef9535dec8f4b0ab2b503f2b4d507198afc",
            "yi": "0x0667f9dad6aafa5745043addf652ea155885c7b7e88dca296b6e2cc96c475098",
            "yr": "0x21a73b10c7523524cbc37d791c730ce12b5fe7c1031da4d899bffad808b7b305"
        },
        "header": {
            "baseFee": "0x174876e800",
            "bloom": "0x10040000000000000000000000000000000000000000000000000000081020010000000080800000000000000000000000000000000008800000000004000000100000200000000000000008000000000000000000800000000000000000000000008000020000000000000100000800000000000a00000000000010000008000000000000000000000000000000000000000000000000000000200000000000000000000000000080002000000080020000000000800000000000000000000000000002000002000000000100000000000080000000010000000000000020000400000000000000000000000000000000004000000000004000000000000000",
            "coinbase": "0x7A3a26123DBD9CFeFc1725fe7779580B987251Cb",
            "extra": "0xd7820304846765746888676f312e31352e36856c696e75780000000000000000f8d3c0c0c080b841ec94af426b69d37aa5b3da5015b6920b9dd21021c7d3867c3fe6df1dd07530b3239ac9ad4055971426ae6415bb9fd0b953dcb0e506e45856fcc0debc31c7336901f84407b8401e961ced90d2132b36e9dee49427eff38f46d359dba22c86da66caa064990ab62dc6c3dcaca9fd593f01deba3a81627ca0671c1694241ba35db1e69f894de97f80f8440fb84011a759c98d54d7ed6b3333839f7d88ab5690e2be79b1e06b292cae5baa2f3baf03007c6a2617efc13079d62a86582beb2fdb13069a4586b3ae22da8efe689f9080",
            "gasLimit": "0x7a1200",
            "gasUsed": "0x0",
            "minDigest": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "0x0000000000000000",
            "number": "0x7d0",
            "parentHash": "0x9887e5ccdf1e53f2857ecc31426280a38182617c0e72c91853872dd12f71fc83",
            "receiptHash": "0x0c26c8fa37c6951b3f012347cfcc98f44068e027035a50571e0ae53ed2068b78",
            "root": "0xba396db6fffc6209c941330b925e3b8fa6a1a63c9dca393ac4ae1b2d5504480b",
            "time": "0x62a2f7e8",
            "txHash": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
        }
	}`

	res, err := dailRpc().TransactionSendAwait(
		ctx,
		from,
		endPoint,
		[]action.Action{
			action.NewFunctionCall(MethodOfUpdateBlockHeader, []byte(s), types.DefaultFunctionCallGas, types.Balance{}),
		},
		client.WithLatestBlock(),
		client.WithKeyPair(kp),
	)

	if err != nil {
		t.Errorf("failed to do txn: %s", err)
	}
	t.Logf("sendTx success %v", res)
}

func Test_new(t *testing.T) {
	kp, err := keystore.NearKeyPairFrom("local", "/Users/xm", from)
	if err != nil {
		t.Fatalf("keypair err %v", err)
	}

	ctx := client.ContextWithKeyPair(context.Background(), kp)
	s := `{
    "threshold": 3,
    "epoch": 1,
    "epoch_size": 1000,
    "validators": [
        {
            "address": "0xb4e1bc0856f70a55764fd6b3f8dd27f2162108e9",
            "weight": 1,
            "g1_pub_key": {
                "x": "0x13524ec450b9ac611fb332a25b6c2eb436d13ac8a540f69a50d6ff8d4fe9f249",
                "y": "0x2b7d0f6e80e80e9b5f9c7a9fa2d482c2e8ea6c1657057c5548b7e30412d48bc3"
            },
            "_BLSPublicKey": "0x1446c55bf2cd8f31d31eaa58b081b30cea50b4d8d9096682576688c2f9ae627d27f97e09b64c99b0e49e7c33c864c089ba5c03cf27e04af4ac457edb46d12ce92aa1ca438667203f7d1696549bd861bf6f11cd3cb7d67738222428137ecadac91b68426ad13c8af92a9b8dc62475fbb1617640c635b812733efc9b7d21c8ab49",
            "_BLSG1PublicKey": "0x13524ec450b9ac611fb332a25b6c2eb436d13ac8a540f69a50d6ff8d4fe9f2492b7d0f6e80e80e9b5f9c7a9fa2d482c2e8ea6c1657057c5548b7e30412d48bc3",
            "_UncompressedBLSPublicKey": "FEbFW/LNjzHTHqpYsIGzDOpQtNjZCWaCV2aIwvmuYn0n+X4JtkyZsOSefDPIZMCJulwDzyfgSvSsRX7bRtEs6SqhykOGZyA/fRaWVJvYYb9vEc08t9Z3OCIkKBN+ytrJG2hCatE8ivkqm43GJHX7sWF2QMY1uBJzPvybfSHIq0k="
        },
        {
            "address": "0x7a3a26123dbd9cfefc1725fe7779580b987251cb",
            "weight": 1,
            "g1_pub_key": {
                "x": "0x0e3450c5b583e57d8fe736d276e9e4bb2ce4b38a5e9ac77b1289ba14a5e9cf58",
                "y": "0x1ce786f52d5bd0e77c1eacfa3dd5df0e22464888fa4bfab6eff9f29e8f86084b"
            },
            "_BLSPublicKey": "0x25f8387695e95f4224919051cfebe64efa2efb4ca8b82bd04cada59edb9e54920a10ea0f9064067049aaf12299592b4746fe4085516fc3f00fbc7dbdd1c545fe126f7bd8d6b5eff857e6fa227c58eea0da0984b80922f5bb33dba1832cf8722f26dc97985df5bf1b4db96f102b905c6982192feecf860ffba2d5eb4db3a64bda",
            "_BLSG1PublicKey": "0x0e3450c5b583e57d8fe736d276e9e4bb2ce4b38a5e9ac77b1289ba14a5e9cf581ce786f52d5bd0e77c1eacfa3dd5df0e22464888fa4bfab6eff9f29e8f86084b",
            "_UncompressedBLSPublicKey": "Jfg4dpXpX0IkkZBRz+vmTvou+0youCvQTK2lntueVJIKEOoPkGQGcEmq8SKZWStHRv5AhVFvw/APvH290cVF/hJve9jWte/4V+b6InxY7qDaCYS4CSL1uzPboYMs+HIvJtyXmF31vxtNuW8QK5BcaYIZL+7Phg/7otXrTbOmS9o="
        },
        {
            "address": "0x7607c9cdd733d8cda0a644839ec2bac5fa180ed4",
            "weight": 1,
            "g1_pub_key": {
                "x": "0x2f6dd4eda4296d9cf85064adbe2507901fcd4ece425cc996827ba4a2c111c812",
                "y": "0x1e6fe59e1d18c107d480077debf3ea265a52325725a853a710f7ec3af5e32869"
            },
            "_BLSPublicKey": "0x2c87c9887cd5af4e33cf09b0a7c4840f87b53bfe5a21d9e18051228577fb191615bb9e374b2625f11e47737292a8688c9ce1c8533ab8000f495a4cc2e55f5531303a6a1542be1e2b6a0e6f37fe2f3f2d60d39b7516f831cb9a3811e82f2a2f1825467b24b96006ee5f494e7469b5a599ec6ab885df02d0ead0926ca45c4f2a1b",
            "_BLSG1PublicKey": "0x2f6dd4eda4296d9cf85064adbe2507901fcd4ece425cc996827ba4a2c111c8121e6fe59e1d18c107d480077debf3ea265a52325725a853a710f7ec3af5e32869",
            "_UncompressedBLSPublicKey": "LIfJiHzVr04zzwmwp8SED4e1O/5aIdnhgFEihXf7GRYVu543SyYl8R5Hc3KSqGiMnOHIUzq4AA9JWkzC5V9VMTA6ahVCvh4rag5vN/4vPy1g05t1Fvgxy5o4EegvKi8YJUZ7JLlgBu5fSU50abWlmexquIXfAtDq0JJspFxPKhs="
        },
        {
            "address": "0x65b3fee569bf82ff148bdded9c3793fb685f9333",
            "weight": 1,
            "g1_pub_key": {
                "x": "0x05fde1416ab5b30e4b140ad4a29a52cd9bc85ca27bd4662ba842a2e22118bea6",
                "y": "0x0dc32694f317d886daac5419b39412a33ee89e07d39d557e4e2b0e48696ac311"
            },
            "_BLSPublicKey": "0x138cf64f20414eed4eb6e5b1b0e08b03944d368d2df78461b493c16cee52fca12741fc9a64ca55a6c08987eba75fc1d1e13a64b9e1e6f6a4413acda3b4997adf184ae5b75a5bd807d5d03d698cd0366d443786276749d2831416dd497d3e53170b14f7028a3245c0383e378130ed3c439aff3b29be9195d778c13e462e4a4afa",
            "_BLSG1PublicKey": "0x05fde1416ab5b30e4b140ad4a29a52cd9bc85ca27bd4662ba842a2e22118bea60dc32694f317d886daac5419b39412a33ee89e07d39d557e4e2b0e48696ac311",
            "_UncompressedBLSPublicKey": "E4z2TyBBTu1OtuWxsOCLA5RNNo0t94RhtJPBbO5S/KEnQfyaZMpVpsCJh+unX8HR4TpkueHm9qRBOs2jtJl63xhK5bdaW9gH1dA9aYzQNm1EN4YnZ0nSgxQW3Ul9PlMXCxT3AooyRcA4PjeBMO08Q5r/Oym+kZXXeME+Ri5KSvo="
        }
    ]
}`
	data, _ := json.Marshal(s)
	res, err := dailRpc().TransactionSendAwait(
		ctx,
		from,
		endPoint,
		[]action.Action{
			action.NewFunctionCall("new", data, 0, types.Balance{}),
		},
		client.WithLatestBlock(),
		client.WithKeyPair(kp),
	)

	if err != nil {
		t.Errorf("failed to do txn: %s", err)
	}
	t.Logf("sendTx success %+v", res)
}

func Test_get_sync_header_height(t *testing.T) {
	//args := []byte("{}")
	res, err := dailRpc().ContractViewCallFunction(context.Background(), to, "get_header_height", "e30=", block.FinalityFinal())
	if err != nil {
		t.Fatalf("pack lightNode headerHeight Input failed, err is %v", err.Error())
	}

	if res.Error != nil {
		t.Fatalf("request back resp, exist error, err is %v", res.Error)
	}

	result := &Result{}
	err = json.Unmarshal(res.Result, result)
	if err != nil {
		t.Fatalf("json marshal failed, err is %v, data is %v", err.Error(), string(res.Result))
	}
	t.Logf("resp is ----------- %v", result.Result)
	t.Logf("resp is ----------- %v", string(result.Result))
	height, _ := new(big.Int).SetString(string(result.Result), 10)
	t.Logf("---------------- height %v", height)
}

type Result struct {
	BlockHash   string        `json:"block_hash"`
	BlockHeight int           `json:"block_height"`
	Logs        []interface{} `json:"logs"`
	Result      []byte        `json:"result"`
}
