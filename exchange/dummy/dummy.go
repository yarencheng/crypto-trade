package dummy

import (
	"context"
	"sync"
	"time"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/logger"
)

var log = logger.Get("dummy_exchange.go")

type Dummy struct {
	stopContext context.Context
	stopCancel  context.CancelFunc
	stopWG      sync.WaitGroup
	DelayMs     int64
	LiveOrders  chan<- data.Order
}

func New() *Dummy {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dummy{
		stopContext: ctx,
		stopCancel:  cancel,
	}
}

func (ex *Dummy) Start() {
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
				time.Sleep(time.Duration(ex.DelayMs) * time.Millisecond)
			}
		}
	}()

	log.Infoln("Started")
}

func (ex *Dummy) Finalize() {
	log.Infoln("Stopping")
	ex.stopCancel()
	ex.stopWG.Wait()
	log.Infoln("stopped")
}
