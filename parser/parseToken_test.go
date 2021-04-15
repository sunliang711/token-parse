package parser

import (
	"bytes"
	"testing"
	"token-parse/config"

	"github.com/spf13/viper"
)

func TestParseTransferEvent(t *testing.T) {
	cfg := `
tokens:
  -
    chain: eth-main
    rpc: https://mainnet.infura.io/v3/f26e9265123241a4ba22cb9188089fe5
    rpc1: http://10.1.9.20:8545
    name: USDT
    from_block2: 11538392
    from_block:  7534749
    block_step: 5
    address: 0xdac17f958d2ee523a2206206994597c13d831ec7
    owner: 0x36928500bc1dcd7af6a2b4008875cc336b927d57
    interval: 2
    timeout: 5
`

	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer([]byte(cfg)))
	var tokens []config.TokenConfig
	err := v.UnmarshalKey("tokens", &tokens)
	if err != nil {
		t.Fatalf("UnmarshalKey error: %v", err)
	}
	t.Logf("tokens: %+v", tokens)
	//err = parseTransferEvent(&tokens[0])
	//if err != nil {
	//	t.Logf("parse error: %v", err)
	//}


	parser := New("debug",&tokens[0],"root:root@tcp(127.0.0.1:3306)/tmt?charset=utf8mb4&parseTime=true&loc=Local")
	parser.Start()

}
