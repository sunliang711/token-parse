package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"token-parse/config"
	"token-parse/db"
	"token-parse/parser"
)

func main() {
	configFile := pflag.StringP("config", "c", "config.yaml", "config file")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Panic("read config file error: %v", err)
	}

	var tokens []config.TokenConfig
	err = viper.UnmarshalKey("tokens", &tokens)
	if err != nil {
		logrus.Printf("unmarshal config tokens error: %v", err)
	}

	logrus.Debugf("tokens: %+v", tokens)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	dbOp := db.InitDB(viper.GetString("mode"), viper.GetString("mysql.url"))
	d, err := dbOp.DB()
	if err != nil {
		return
	}
	defer func() {
		logrus.Info("close db")
		d.Close()
	}()

	parsers := make([]*parser.Parser, len(tokens))
	wg := sync.WaitGroup{}
	for i := range tokens {
		go func(i int) {
			wg.Add(1)
			parsers[i] = parser.New(&tokens[i], dbOp, &wg)
			parsers[i].Start()
		}(i)
	}

	select {
	case sig := <-sigCh:
		logrus.Infof("got signal: %v", sig)
		for i := range parsers {
			parsers[i].Stop()
		}
		logrus.Infof("wait for end")
		wg.Wait()
		logrus.Infof("stoped")
	}
}
