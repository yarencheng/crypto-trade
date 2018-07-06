package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yarencheng/gospring/v1"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/exchange/dummy"
	"github.com/yarencheng/crypto-trade/logger"
	"github.com/yarencheng/crypto-trade/strategies"
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
				Config: v1.V(int64(10)),
			},
			{
				Name:   "LiveOrders",
				Config: "orders",
			},
		},
	}, &v1.Bean{
		ID:        "stupid_strategy_1",
		Type:      v1.T(strategies.StupidStrategy{}),
		FactoryFn: strategies.New,
		Properties: []v1.Property{
			{
				Name:   "LiveOrders",
				Config: "ordersBroadcast",
			},
		},
	}, &v1.Bean{
		ID:        "stupid_strategy_2",
		Type:      v1.T(strategies.StupidStrategy{}),
		FactoryFn: strategies.New,
		Properties: []v1.Property{
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

	_, err = ctx.GetByID("stupid_strategy_1")
	if err != nil {
		log.Fatalf("Failed to get stupid_strategy_1. err: [%v]", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	s := <-stop
	log.Infof("Stopped by signal [%v].\n", s)

	b, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wait := make(chan int, 1)
	go func() {
		err = ctx.Stop(b)
		if err != nil {
			log.Fatalf("Stop ctx failed. err: [%v]", err)
		}
		wait <- 1
	}()

	select {
	case <-b.Done():
	case <-wait:
	}

	if err := b.Err(); err != nil {
		log.Fatalf("Stop failed. err: [%v]", err)
	}

	log.Infof("Stopped.")
}
