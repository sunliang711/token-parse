package tools

import "fmt"

const (
	ETH_ADDRESS_LEN = 40
)

func AddressFormat(address string) string {
	if len(address) > ETH_ADDRESS_LEN {
		return fmt.Sprintf("0x%s", address[len(address)-ETH_ADDRESS_LEN:])
	} else {
		return address
	}
}
