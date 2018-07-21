package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
	"github.com/yarencheng/crypto-trade/go/reproducer"
	"github.com/yarencheng/crypto-trade/go/strategies"
	"github.com/yarencheng/gospring/applicationcontext"
	"github.com/yarencheng/gospring/v1"
)

func main() {

	ctx := applicationcontext.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:        "reproducer",
		Type:      v1.T(reproducer.SqliteReproducer{}),
		FactoryFn: reproducer.New,
		Properties: []v1.Property{
			{
				Name:   "OrderBooks",
				Config: "in_orders",
			}, {
				Name:   "BuyOrders",
				Config: "out_orders",
			}, {
				Name:   "Path",
				Config: v1.V("all.sqlite"),
			},
		},
	}, &v1.Bean{
		ID:        "stupid_strategy",
		Type:      v1.T(strategies.StupidStrategy{}),
		FactoryFn: strategies.NewStupidStrategy,
		Properties: []v1.Property{
			{
				Name:   "InOrders",
				Config: "in_orders",
			}, {
				Name:   "OutOrders",
				Config: "out_orders",
			},
		},
	}, &v1.Channel{
		ID:   "in_orders",
		Type: v1.T(entity.OrderBookEvent{}),
		Size: 1000,
	}, &v1.Channel{
		ID:   "out_orders",
		Type: v1.T(entity.BuyOrderEvent{}),
		Size: 1000,
	})

	if err != nil {
		logger.Fatalf("Add configs to ctx failed. err: [%v]", err)
	}

	_, err = ctx.GetByID("reproducer")
	if err != nil {
		logger.Fatalf("Failed to get reproducer. err: [%v]", err)
	}

	_, err = ctx.GetByID("stupid_strategy")
	if err != nil {
		logger.Fatalf("Failed to get stupid_strategy. err: [%v]", err)
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
