package cmd

import (
	"github.com/mapprotocol/compass/chain_tools"
	"github.com/mapprotocol/compass/cmd/cmd_runtime"
	"github.com/mapprotocol/compass/utils"
	"github.com/spf13/cobra"
)

var (
	passwordCmd = &cobra.Command{
		Use:   "password",
		Short: "Get account info.",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig, _, _ := cmd_runtime.ReadTomlConfig()

			_, correctPassword := chain_tools.LoadPrivateKey(globalConfig.Keystore, "")
			println("Set next line to config.toml global.password :")
			println(utils.AesCbcEncrypt([]byte(correctPassword)))
		},
	}
)
