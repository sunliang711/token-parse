package config

// TokenConfig stores ERC20 token config info
type TokenConfig struct {
	RPC       string
	TokenName string `mapstructure:"token_name"`
	// big.Int? OR string?
	//FromBlock string `mapstructure:"from_block"`
	FromBlock uint `mapstructure:"from_block"` // mapstructure tag for viper
	BlockStep uint `mapstructure:"block_step"`
	// contract address where log originate
	ContractAddress string `mapstructure:"contract_address"`
	Owner           string

	// poll interval
	Interval uint
	Timeout  uint
}
