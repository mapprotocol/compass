package cmd_runtime

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mapprotocol/compass/chains"
	"github.com/mapprotocol/compass/chains/ethereum"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	globalConfig, srcChainConfig, dstChainConfig := ReadTomlConfig()

	keystore := globalConfig.Keystore
	password := globalConfig.Password

	SrcInstance = ethereum.NewEthChain(
		srcChainConfig.Name, srcChainConfig.ChainId,
		srcChainConfig.BlockCreatingTime, srcChainConfig.RpcUrl,
		srcChainConfig.StableBlock,
		"", "",
	)

	DstInstance = ethereum.NewEthChain(
		dstChainConfig.Name, dstChainConfig.ChainId,
		dstChainConfig.BlockCreatingTime, dstChainConfig.RpcUrl,
		dstChainConfig.StableBlock,
		dstChainConfig.RelayerContractAddress, dstChainConfig.HeaderStoreContractAddress,
	)

	if keystore == "" {
		log.Fatal("keystore is not set correctly at config.toml.")
	}
	if !strings.Contains(keystore, "/") && !strings.Contains(keystore, "\\") {
		keystore = filepath.Join(filepath.Dir(os.Args[0]), keystore)
	}
	if !common.FileExist(keystore) {
		log.Fatal("keystore file not exists.")

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
