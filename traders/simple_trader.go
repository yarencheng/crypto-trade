package traders

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data/currencies"
	"github.com/yarencheng/crypto-trade/exchanges"
)

type SimpleTrader struct {
	exchanges []exchanges.Exchange
	stop      chan int
}

func NewSimpleTrader() *SimpleTrader {

	trader := &SimpleTrader{}

	trader.stop = make(chan int)

	return trader
}

func (trader *SimpleTrader) AddExchange(exchange exchanges.Exchange) {
	trader.exchanges = append(trader.exchanges, exchange)
}

func (trader *SimpleTrader) Start() error {

	ctx, cancel := context.WithCancel(context.Background())

	for _, ex := range trader.exchanges {

		go func() {
			orders, _ := ex.GetOrders(currencies.BTC, currencies.ETH)
			for {
				select {
				case order := <-orders:
					logrus.Infoln("Get order", order)
				case <-ctx.Done():
					break
				}
			}
		}()
	}

	go func() {
		<-trader.stop
		cancel()
	}()

	return nil
}

func (trader *SimpleTrader) Stop() error {
	trader.stop <- 0

	return nil
}
