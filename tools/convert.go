package tools

import (
	"strconv"
	"strings"
)

func HexString2Uint(numStr string) (uint64, error) {
	if strings.HasPrefix(numStr,"0x"){
		numStr = numStr[2:]
	}
	return strconv.ParseUint(numStr, 16, 64)
}
