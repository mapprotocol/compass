package common

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/mapprotocol/compass/libs/sync_libs/chain_structs"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

type waitTimeAndMessage struct {
	Time    time.Duration
	Message string
}

var (
	DstInstance chain_structs.ChainInterface
	SrcInstance chain_structs.ChainInterface

	StructRegisterNotRelayer = &waitTimeAndMessage{
		Time:    2 * time.Minute,
		Message: "registered not relayer",
	}
	StructUnregistered = &waitTimeAndMessage{
		Time:    10 * time.Minute,
		Message: "Unregistered",
	}
	StructUnStableBlock = &waitTimeAndMessage{
		Time:    time.Second * 2, //it will update at InitClient func
		Message: "Unstable block",
	}
)

func InitClient() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error ! loading .env file")
	}
	srcChainIdStr := os.Getenv("src_chain_enum")
	dstChainIdStr := os.Getenv("dst_chain_enum")
	keystore := os.Getenv("keystore")
	password := os.Getenv("password")
	var chainEnumIntSrc, chainEnumIntDst int
	var ok bool
	if srcChainIdStr == "" {
		println("src_chain_enum not be set at .env")
		os.Exit(1)
	}
	if srcChainIdStr == dstChainIdStr {
		println("src_chain_enum and dst_chain_enum are not allowed the same")
		os.Exit(1)
	}
	chainEnumIntSrc, _ = strconv.Atoi(srcChainIdStr)
	if SrcInstance, ok = chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(chainEnumIntSrc)]; !ok {
		println("src_chain_enum not be set correctly at .env")
		os.Exit(1)
	}
	if dstChainIdStr == "" {
		println("dst_chain_enum not be set at .env")
		os.Exit(1)
	}
	chainEnumIntDst, _ = strconv.Atoi(dstChainIdStr)
	if DstInstance, ok = chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(chainEnumIntDst)]; !ok {
		println("dst_chain_enum not be set correctly at .env")
		os.Exit(1)
	}

	if keystore == "" || !common.FileExist(keystore) {
		println("keystore be set correctly at .env, file not exists.")
		os.Exit(1)
	}
	DstInstance.SetTarget(keystore, password)
	StructUnStableBlock.Time = SrcInstance.NumberOfSecondsOfBlockCreationTime()
}
func DisplayMessageAndSleep(s *waitTimeAndMessage) {
	log.Println(s.Message)
	time.Sleep(s.Time)
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
