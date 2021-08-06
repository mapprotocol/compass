package sync_cmd

import (
	"bufio"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"log"
	"os"
	"signmap/libs"
	"signmap/libs/sync_libs/chain_structs"
	"strconv"
	"strings"
)

var (
	envFile   = ".env"
	cmdConfig = &cobra.Command{
		Use:   "config",
		Short: "Configure the application. Create or update " + envFile + " file.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := cmd.Help()
			if err != nil {
				return
			}
		},
	}
	configGet = &cobra.Command{
		Use:   "get ",
		Short: "Read .env file content",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			readEnvFileContents()
		},
	}
	configSet = &cobra.Command{
		Use:   "set",
		Short: "Create or update .env file.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if common.FileExist(envFile) {
				//  print alert info
				readEnvFileContents()
				print(".env file already exists,OverWrite or not (y/n): ")
				input := libs.ReadString()
				if strings.ToLower(input) != "y" {
					return
				}
			}
			var srcChainId, dstChainId int
			var srcChainIdStr, dstChainIdStr, keystorePath, password, input string
			var passwordByte []byte
			var err error

			fileContents := ""

			println("ChainInterface list: ")
			printMapOption()
			for {
				print("Select source chain id:")
				srcChainIdStr = libs.ReadString()
				srcChainId, _ = strconv.Atoi(srcChainIdStr)
				if _, ok := chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(srcChainId)]; ok {
					break
				}
			}
			fileContents += "src_chain_enum=" + srcChainIdStr + "\n"
			for {
				print("Select  target ChainInterface id:")
				dstChainIdStr = libs.ReadString()
				dstChainId, _ = strconv.Atoi(dstChainIdStr)
				if _, ok := chain_structs.ChainEnum2Instance[chain_structs.ChainEnum(dstChainId)]; ok && dstChainId != srcChainId {
					break
				}
			}
			fileContents += "dst_chain_enum=" + dstChainIdStr + "\n"
			for {
				println("Enter the keystore file path.")
				keystorePath = libs.ReadString()
				if common.FileExist(keystorePath) {
					break
				}
				println(keystorePath, "file not exists.")
			}
			fileContents += "keystore=" + keystorePath + "\n"
			println("Enter the password .For safety, it can be empty,but it can't be wrong. If it is empty，enter the password manually when required")
			passwordByte, err = terminal.ReadPassword(0)
			if err != nil {
				log.Println("Read password  error : ", err)
			}
			password = string(passwordByte)
			fileContents += "password=" + password + "\n"
			println("The new configuration is as follows")
			println(fileContents)
			print("Confirm the change，Make sure the password is correct or empty.(y/n):")
			input = libs.ReadString()
			if strings.ToLower(input) == "y" {
				err = ioutil.WriteFile(envFile, []byte(fileContents), 0600)
				if err != nil {
					log.Fatal("write env file error: ", err)
				}
				println("Successful.")
			} else {
				println("Nothing change.")
			}
		},
	}
)

func readEnvFileContents() bool {
	file, err := os.Open(envFile)
	if err != nil {
		println(envFile, " file does not exist.")
		return false
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	return true
}
func printMapOption() {
	for k, v := range chain_structs.ChainEnum2Instance {
		println(k, ":", v.GetName())
	}
}
func cmdConfigFunc() *cobra.Command {
	cmdConfig.AddCommand(configGet, configSet)
	return cmdConfig
}
