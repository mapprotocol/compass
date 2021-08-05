package sync_cmd

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"log"
	"os"
	"signmap/libs/sync_libs/chain_structs"
	"strconv"
)

var (
	srcInstance, dstInstance chain_structs.MapChain
)

func initClient() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error ! loading .env file")
	}
	srcChainIdStr := os.Getenv("src_chain_enum")
	dstChainIdStr := os.Getenv("dst_chain_enum")
	keystore := os.Getenv("keystore")
	password := os.Getenv("password")
	var chainEnumInt int
	var ok bool
	if srcChainIdStr == "" {
		println("src_chain_enum not be set at .env")
		os.Exit(1)
	}
	if srcChainIdStr == dstChainIdStr {
		println("src_chain_enum and dst_chain_enum are not allowed the same")
		os.Exit(1)
	}
	chainEnumInt, _ = strconv.Atoi(srcChainIdStr)
	if srcInstance, ok = chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(chainEnumInt)]; !ok {
		println("src_chain_enum not be set correctly at .env")
		os.Exit(1)
	}
	if dstChainIdStr == "" {
		println("dst_chain_enum not be set at .env")
		os.Exit(1)
	}
	chainEnumInt, _ = strconv.Atoi(srcChainIdStr)
	if dstInstance, ok = chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(chainEnumInt)]; !ok {
		println("dst_chain_enum not be set correctly at .env")
		os.Exit(1)
	}

	if keystore == "" || !common.FileExist(keystore) {
		println("keystore be set correctly at .env, file not exists.")
		os.Exit(1)
	}
	dstInstance.SetTarget(keystore, password)
}
