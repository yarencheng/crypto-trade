package traders

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/exchanges"
)

type SimpleTrader struct {
	Exchanges []exchanges.ExchangeI
	stop      chan int
}

func NewSimpleTrader() *SimpleTrader {

	trader := &SimpleTrader{}

	trader.stop = make(chan int)

	return trader
}

func (trader *SimpleTrader) String() string {
	return fmt.Sprintf("SimpleTrader[@%p]", trader)
}

func (trader *SimpleTrader) AddExchange(exchange exchanges.ExchangeI) {
	trader.Exchanges = append(trader.Exchanges, exchange)
}

func (trader *SimpleTrader) Start() error {

	ctx, cancel := context.WithCancel(context.Background())

	for _, ex := range trader.Exchanges {

		go func() {
			orders, _ := ex.GetOrders(data.BTC, data.ETH)
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
