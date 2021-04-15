package tools

import (
	"context"
	"time"
)

func GetContextDefault() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	return ctx
}
