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
	LiveOrders  chan<- data.Order
}

func NewDummyExchange(delayMs int64) *DummyExchange {
	ctx, cancel := context.WithCancel(context.Background())
	return &DummyExchange{
		stopContext: ctx,
		stopCancel:  cancel,
		delayMs:     delayMs,
	}
}

func (ex *DummyExchange) Init() {
	log.Infoln("Starting")

	go func() {
		defer ex.stopWG.Done()
		for {
			order := data.Order{
				From: data.BTC,
				To:   data.ETH,
			}
			select {
			case <-ex.stopContext.Done():
				return
			case ex.LiveOrders <- order:
				log.Infoln("Send order", order)
				time.Sleep(time.Duration(ex.delayMs) * time.Millisecond)
			}
		}
	}()

	log.Infoln("Started")
}

func (ex *DummyExchange) Finalize() {
	log.Infoln("Stopping")
	ex.stopCancel()
	ex.stopWG.Wait()
	log.Infoln("stopped")
}
