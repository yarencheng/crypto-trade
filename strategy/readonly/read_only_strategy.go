package readonly

import (
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
)

type ReadOnlyStrategy struct {
	ordersMutex       sync.Mutex
	orders            chan data.Order
	tradeResultsMutex sync.Mutex
	tradeResults      chan data.TradeResult
	trades            chan data.Trade
	stops             chan int
}

func NewReadOnlyStrategy() *ReadOnlyStrategy {

	var s ReadOnlyStrategy

	s.trades = make(chan data.Trade, 10)
	s.stops = make(chan int)

	return &s
}

func (strategy *ReadOnlyStrategy) ReadOrders(orders chan data.Order) {

	strategy.ordersMutex.Lock()

	strategy.orders = orders

	strategy.ordersMutex.Unlock()
}

func (strategy *ReadOnlyStrategy) ReadTradeResults(results chan data.TradeResult) {

	strategy.ordersMutex.Lock()

	strategy.tradeResults = results

	strategy.ordersMutex.Unlock()
}

func (strategy *ReadOnlyStrategy) WriteTrades() chan data.Trade {
	return strategy.trades
}

func (strategy *ReadOnlyStrategy) Run() {

	go func() {
		logrus.Infoln("ReadOnlyStrategy is running")
		for {
			select {
			case order := <-strategy.orders:

				logrus.Infoln("ReadOnlyStrategy read an order", order)

			case tradeResult := <-strategy.tradeResults:

				logrus.Infoln("ReadOnlyStrategy read a trade result", tradeResult)

			case <-strategy.stops:

				logrus.Infoln("ReadOnlyStrategy stops")

				break
			}
		}
	}()

}

func (strategy *ReadOnlyStrategy) Stop() {
	strategy.stops <- 0
}
