package sync_cmd

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/exec"
	"runtime"
	"signmap/libs/sync_libs/chain_structs"
	"strconv"
)

var (
	srcInstance, dstInstance chain_structs.ChainInterface
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

var clear map[string]func() //create a map for storing clear funcs

func init() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
	clear["darwin"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			return
		}
	}
}

func CallClear() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}
