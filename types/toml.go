package types

type GlobalConfig struct {
	Keystore                string `toml:"keystore"`
	Password                string `toml:"password"`
	BlockNumberByEstimation bool   `toml:"block_number_by_estimation"`
	StartWithBlock          int64  `toml:"start_with_block"`
}
type ChainConfig struct {
	Name                       string  `toml:"name"`
	ChainId                    ChainId `toml:"chain_id"`
	BlockCreatingTime          int     `toml:"block_creating_seconds"`
	RpcUrl                     string  `toml:"rpc_url"`
	StableBlock                uint64  `toml:"stable_block"`
	RelayerContractAddress     string  `toml:"relayer_contract_address"`
	HeaderStoreContractAddress string  `toml:"header_store_contract_address"`
	RouterContractAddress      string  `toml:"router_contract_address"`
}
