package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/yarencheng/gospring/v1"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/exchange/dummy"
	"github.com/yarencheng/crypto-trade/logger"
	"github.com/yarencheng/gospring/application_context"
)

var log = logger.Get("main.go")

func main() {

	ctx := application_context.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:        "dummy_exchange",
		Type:      v1.T(dummy.Dummy{}),
		FactoryFn: dummy.New,
		Properties: []v1.Property{
			{
				Name:   "DelayMs",
				Config: v1.V(int64(1000)),
			},
			{
				Name:   "LiveOrders",
				Config: "ordersBroadcast",
			},
		},
	}, &v1.Channel{
		ID:   "orders",
		Type: v1.T(data.Order{}),
		Size: 10,
	}, &v1.Broadcast{
		ID:       "ordersBroadcast",
		SourceID: "orders",
		Size:     10,
	})

	if err != nil {
		log.Fatalf("Add configs to ctx failed. err: [%v]", err)
	}

	_, err = ctx.GetByID("dummy_exchange")
	if err != nil {
		log.Fatalf("Failed to get dummy_exchange. err: [%v]", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	s := <-stop

	err = ctx.Stop(context.Background())
	if err != nil {
		log.Fatalf("Stop ctx failed. err: [%v]", err)
	}

	log.Infof("Stopped by signal [%v].\n", s)
}
