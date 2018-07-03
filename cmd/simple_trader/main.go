package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/yarencheng/gospring/v1"

	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/logger"
	"github.com/yarencheng/crypto-trade/strategies"
	"github.com/yarencheng/crypto-trade/traders"
	gs "github.com/yarencheng/gospring"
	"github.com/yarencheng/gospring/application_context"
)

var log = logger.Get("main.go")

func main1() {

	beans := gs.Beans(
		gs.Bean(traders.SimpleTrader{}).
			ID("trader").
			Singleton().
			Factory(traders.NewSimpleTrader).
			Property("Exchanges", gs.Ref("dummy_exchange")).
			Property("Strategy", gs.Ref("stupid_strategy")),
		gs.Bean(exchanges.DummyExchange{}).
			ID("dummy_exchange").
			Singleton().
			Factory(exchanges.NewDummyExchange, int64(1000)),
		gs.Bean(strategies.StupidStrategy{}).
			ID("stupid_strategy").
			Singleton().
			Factory(strategies.NewStupidStrategy),
	)

	ctx, e := gs.NewApplicationContext(beans...)
	if e != nil {
		log.Errorln("Can't create gospring contex. Caused by:", e)
	}
	defer ctx.Finalize()

	_, e = ctx.GetBean("trader")

	if e != nil {
		log.Errorln("Can't create trader. Caused by:", e)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Infoln("Stop trader")
}

func main() {

	ctx := application_context.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:          "dummy_exchange",
		Type:        v1.T(exchanges.DummyExchange{}),
		FactoryFn:   exchanges.NewDummyExchange,
		FactoryArgs: []interface{}{int64(1000)},
	})

	if err != nil {
		log.Fatalf("Add configs to ctx failed. err: [%v]", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	s := <-stop
	log.Infoln("Stopped by signal [%v]", s)
}
