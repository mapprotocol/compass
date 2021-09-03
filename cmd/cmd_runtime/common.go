package cmd_runtime

import (
	"github.com/mapprotocol/compass/chains/ethereum"
	"github.com/mapprotocol/compass/types"
	"github.com/mapprotocol/compass/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func InitClient() {
	ReadTomlConfig()

	keystore := GlobalConfigV.Keystore
	password := GlobalConfigV.Password
	BlockNumberByEstimation = GlobalConfigV.BlockNumberByEstimation

	SrcInstance = ethereum.NewEthChain(
		SrcChainConfig.Name, SrcChainConfig.ChainId,
		SrcChainConfig.BlockCreatingTime, SrcChainConfig.RpcUrl,
		SrcChainConfig.StableBlock,
		"", "",
	)

	DstInstance = ethereum.NewEthChain(
		DstChainConfig.Name, DstChainConfig.ChainId,
		DstChainConfig.BlockCreatingTime, DstChainConfig.RpcUrl,
		DstChainConfig.StableBlock,
		DstChainConfig.RelayerContractAddress, DstChainConfig.HeaderStoreContractAddress,
	)

	if keystore == "" {
		log.Fatal("keystore is not set correctly at config.toml.")
	}
	if !strings.Contains(keystore, "/") && !strings.Contains(keystore, "\\") {
		keystore = filepath.Join(filepath.Dir(os.Args[0]), keystore)
	}
	if password != "" {
		password = string(utils.AesCbcDecrypt(password))
	}
	DstInstance.SetTarget(keystore, password)
	StructUnStableBlock.Time = SrcInstance.NumberOfSecondsOfBlockCreationTime()
}
func DisplayMessageAndSleep(s *types.WaitTimeAndMessage) {
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
