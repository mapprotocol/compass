package cmd_runtime

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/mapprotocol/compass/chains"
	log "github.com/sirupsen/logrus"
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
	DstInstance chains.ChainInterface
	SrcInstance chains.ChainInterface

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
	srcChainIdStr := os.Getenv("src_chain_id")
	dstChainIdStr := os.Getenv("dst_chain_id")
	keystore := os.Getenv("keystore")
	password := os.Getenv("password")
	var chainEnumIntSrc, chainEnumIntDst int
	var ok bool
	if srcChainIdStr == "" {
		log.Fatal("src_chain_id not be set at .env.")
	}
	if srcChainIdStr == dstChainIdStr {
		log.Fatal("src_chain_id and dst_chain_id are not allowed the same.")
	}
	chainEnumIntSrc, _ = strconv.Atoi(srcChainIdStr)
	if SrcInstance, ok = ChainEnum2Instance[chains.ChainEnum(chainEnumIntSrc)]; !ok {
		log.Fatal("src_chain_id not be set correctly at .env.")
	}
	if dstChainIdStr == "" {
		log.Fatal("dst_chain_id is not set at .env.")
	}
	chainEnumIntDst, _ = strconv.Atoi(dstChainIdStr)
	if DstInstance, ok = ChainEnum2Instance[chains.ChainEnum(chainEnumIntDst)]; !ok {
		log.Fatal("dst_chain_id is not set correctly at .env")
	}

	if keystore == "" || !common.FileExist(keystore) {
		log.Fatal("keystore is not set correctly at .env, file not exists.")
	}
	DstInstance.SetTarget(keystore, password)
	StructUnStableBlock.Time = SrcInstance.NumberOfSecondsOfBlockCreationTime()
}
func DisplayMessageAndSleep(s *waitTimeAndMessage) {
	log.Infoln(s.Message)
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
