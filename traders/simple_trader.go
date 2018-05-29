package traders

import (
	"fmt"

	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/logger"
	"github.com/yarencheng/crypto-trade/strategies"
)

// var log = logrus.New().WithField("file", "simple_trader.go")
var log = logger.Get("simple_trader.go")

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

	// ctx, cancel := context.WithCancel(context.Background())

	// for _, ex := range trader.Exchanges {
	// 	orders, _ := ex.GetOrders(data.BTC, data.ETH)

	// 	go func() {
	// 		for {
	// 			select {
	// 			case order := <-orders:
	// 				log.Infoln("Get order", order)
	// 			case <-ctx.Done():
	// 				return
	// 			}
	// 		}
	// 	}()
	// }

	// go func() {
	// 	<-trader.stop
	// 	cancel()
	// }()
	return nil
}

func (trader *SimpleTrader) Finalize() error {

	trader.stop <- 0

	return nil
}
