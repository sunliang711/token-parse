package config

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
)

func TestParseMemory(t *testing.T) {
	cfg := `
tokens:
  -
    chain: eth-main
    rpc: https://mainnet.infura.io/v3/f26e9265123241a4ba22cb9188089fe5
    name: USDT
    from_block: 4634748
    address: 0xdac17f958d2ee523a2206206994597c13d831ec7
    owner: 0x36928500bc1dcd7af6a2b4008875cc336b927d57
    interval: 10
    timeout: 5
`
	v := viper.New()
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer([]byte(cfg)))
	var tokens []TokenConfig
	err := v.UnmarshalKey("tokens", &tokens)
	if err != nil {
		t.Fatalf("UnmarshalKey error: %v", err)
	}
	t.Logf("tokens: %+v", tokens)
}

func TestParseFile(t *testing.T) {
	v := viper.New()
	v.SetConfigFile("config.yaml")
	err := v.ReadInConfig()
	if err != nil {
		t.Fatalf("Read config error: %v", err)
	}
	var tokens []TokenConfig
	err = v.UnmarshalKey("tokens", &tokens)
	if err != nil {
		t.Fatalf("UnmarshalKey error: %v", err)
	}
	t.Logf("tokens: %+v", tokens)

}
