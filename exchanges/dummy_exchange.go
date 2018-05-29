package exchanges

import (
	"context"
	"sync"
	"time"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/logger"
)

var log = logger.Get("dummy_exchange.go")

type DummyExchange struct {
	stopContext context.Context
	stopCancel  context.CancelFunc
	stopWG      sync.WaitGroup
	delayMs     int64
}

func NewDummyExchange(delayMs int64) *DummyExchange {
	ctx, cancel := context.WithCancel(context.Background())
	return &DummyExchange{
		stopContext: ctx,
		stopCancel:  cancel,
		delayMs:     delayMs,
	}
}

func (ex *DummyExchange) GetName() string {
	return "Dummy Exchange"
}

func (ex *DummyExchange) Ping() float64 {
	return 0
}

func (ex *DummyExchange) Finalize() {
	log.Infoln("Stopping")
	ex.stopCancel()
	ex.stopWG.Wait()
	log.Infoln("stopped")
}

func (ex *DummyExchange) GetOrders(from data.Currency, to data.Currency) (<-chan data.Order, error) {

	c := make(chan data.Order, 10)
	ex.stopWG.Add(1)

	go func() {
		defer close(c)
		defer ex.stopWG.Done()
		for {
			order := data.Order{
				From: data.BTC,
				To:   data.ETH,
			}
			select {
			case <-ex.stopContext.Done():
				return
			case c <- order:
				log.Infoln("Send order", order)
				log.Debugln("Send order", order)
				time.Sleep(time.Duration(ex.delayMs) * time.Millisecond)
			}
		}
	}()

	return c, nil
}

func (ex *DummyExchange) Buy(from data.Order) chan data.TradeResult {
	c := make(chan data.TradeResult)
	c <- data.TradeResult{IsSucess: true}
	return c
}
