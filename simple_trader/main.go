package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/traders"
	gs "github.com/yarencheng/gospring"
)

func init() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
}

var log = logrus.New()

func main() {
	beans := gs.Beans(
		gs.Bean(traders.SimpleTrader{}).
			ID("trader").
			Singleton().
			Init("Start").
			Finalize("Stop").
			Factory(traders.NewSimpleTrader).
			Property("Exchanges", gs.Ref("dummy_exchange")),
		gs.Bean(exchanges.DummyExchange{}).
			ID("dummy_exchange").
			Singleton(),
	)

	ctx, e := gs.NewApplicationContext(beans...)
	defer ctx.Finalize()

	if e != nil {
		logrus.Errorln("Can't create gospring contex. Caused by:", e)
	}

	_, e = ctx.GetBean("trader")

	if e != nil {
		logrus.Errorln("Can't create trader. Caused by:", e)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	logrus.Infoln("Stop trader")
}
