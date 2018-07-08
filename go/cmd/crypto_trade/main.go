package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/entity/recorder/sqlite"
	"github.com/yarencheng/crypto-trade/go/exchange/bitfinex"
	"github.com/yarencheng/crypto-trade/go/logger"
	"github.com/yarencheng/gospring/applicationcontext"
	"github.com/yarencheng/gospring/v1"
)

func main() {

	ctx := applicationcontext.Default()

	err := ctx.AddConfigs(&v1.List{
		ID:   "monitor_crypto_currencies",
		Type: v1.T(entity.Currency("")),
		Configs: []interface{}{
			v1.V(entity.BTC),
			v1.V(entity.ETH),
		},
	}, &v1.Bean{
		ID:        "bitfinex",
		Type:      v1.T(bitfinex.Bitfinex{}),
		FactoryFn: bitfinex.New,
		Properties: []v1.Property{
			{
				Name:   "OrderBooks",
				Config: "orders",
			}, {
				Name:   "Currencies",
				Config: "monitor_crypto_currencies",
			},
		},
	}, &v1.Bean{
		ID:        "sqlite_recorder",
		Type:      v1.T(sqlite.Sqlite{}),
		FactoryFn: sqlite.New,
		Properties: []v1.Property{
			{
				Name:   "OrderBooks",
				Config: "orders",
			}, {
				Name:   "Path",
				Config: v1.V("order.sqlite"),
			},
		},
	}, &v1.Channel{
		ID:   "orders",
		Type: v1.T(entity.OrderBook{}),
		Size: 10,
	})

	if err != nil {
		logger.Fatalf("Add configs to ctx failed. err: [%v]", err)
	}

	_, err = ctx.GetByID("sqlite_recorder")
	if err != nil {
		logger.Fatalf("Failed to get sqlite_recorder. err: [%v]", err)
	}

	_, err = ctx.GetByID("bitfinex")
	if err != nil {
		logger.Fatalf("Failed to get bitfinex. err: [%v]", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	s := <-stop
	logger.Infof("Stopped by signal [%v].\n", s)

	b, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wait := make(chan int, 1)
	go func() {
		err = ctx.Stop(b)
		if err != nil {
			logger.Fatalf("Stop ctx failed. err: [%v]", err)
		}
		wait <- 1
	}()

	select {
	case <-b.Done():
	case <-wait:
	}

	if err := b.Err(); err != nil {
		logger.Fatalf("Stop failed. err: [%v]", err)
	}

	logger.Infof("Stopped.")
}
