package trader

import (
	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/currency"
	"github.com/yarencheng/crypto-trade/exchange"
)

type SimpleTrader struct {
	exchanges []*exchange.Exchange
	stop      chan int
}

func NewSimpleTrader() *SimpleTrader {

	trader := &SimpleTrader{}

	trader.stop = make(chan int)

	return trader
}

func (trader *SimpleTrader) AddExchange(exchange *exchange.Exchange) {
	trader.exchanges = append(trader.exchanges, exchange)
}

func (trader *SimpleTrader) Start() {
	for _, ex := range trader.exchanges {

		go func() {
			for order := range (*ex).GetOrders(currency.BTC, currency.ETH) {
				logrus.Infoln("Get order", order)
			}
		}()
	}

}

func (trader *SimpleTrader) Stop() {
	trader.stop <- 0
}
