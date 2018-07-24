package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yarencheng/crypto-trade/go/analyzer"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
	"github.com/yarencheng/crypto-trade/go/strategies"
	"github.com/yarencheng/gospring/applicationcontext"
	"github.com/yarencheng/gospring/v1"
)

func main() {

	ctx := applicationcontext.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:        "analyzer",
		Type:      v1.T(analyzer.Analyzer{}),
		FactoryFn: analyzer.New,
		Properties: []v1.Property{
			{
				Name:   "OrderBooks",
				Config: "in_orders",
			}, {
				Name:   "BuyOrders",
				Config: "out_orders",
			}, {
				Name:   "OrderBookPath",
				Config: v1.V("all.sqlite"),
			}, {
				Name:   "ResultPath",
				Config: v1.V("result.sqlite"),
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
		Size: 1,
	}, &v1.Channel{
		ID:   "out_orders",
		Type: v1.T(entity.BuyOrderEvent{}),
		Size: 1,
	})

	if err != nil {
		logger.Fatalf("Add configs to ctx failed. err: [%v]", err)
	}

	_, err = ctx.GetByID("analyzer")
	if err != nil {
		logger.Fatalf("Failed to get analyzer. err: [%v]", err)
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
