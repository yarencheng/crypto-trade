package traders

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/strategies"
)

type SimpleTrader struct {
	Exchanges []exchanges.ExchangeI
	Strategy  strategies.StrategyI
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

func (trader *SimpleTrader) Init() error {

	ctx, cancel := context.WithCancel(context.Background())

	for _, ex := range trader.Exchanges {
		go func() {
			pipe := make(chan data.Order, 10)
			trader.Strategy.In(pipe)
			orders, _ := ex.GetOrders(data.BTC, data.ETH)

			for {
				select {
				case order := <-orders:
					//logrus.Debugln("Get order", order)
					pipe <- order
				case <-ctx.Done():
					// close(pipe)
					break
				}
			}
		}()
	}

	go func() {
		buys, _ := trader.Strategy.Out()

		for {
			select {
			case buy := <-buys:
				logrus.Infoln("Buy order", buy)
				for _, ex := range trader.Exchanges {
					ex.Buy(buy)
				}
			case <-ctx.Done():
				break
			}
		}
	}()

	go func() {
		<-trader.stop
		cancel()
	}()
	return nil
}

func (trader *SimpleTrader) Finalize() error {

	trader.stop <- 0

	return nil
}
