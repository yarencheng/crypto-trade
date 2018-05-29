package strategies

import (
	"context"
	"sync"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/logger"
)

var log = logger.Get("stupid_strategy.go")

type StupidStrategy struct {
	stopContext context.Context
	stopCancel  context.CancelFunc
	stopWG      sync.WaitGroup
	LiveOrders  <-chan data.Order
}

func NewStupidStrategy() *StupidStrategy {
	ctx, cancel := context.WithCancel(context.Background())
	return &StupidStrategy{
		stopContext: ctx,
		stopCancel:  cancel,
	}
}

func (st *StupidStrategy) Init() {
	log.Infoln("Starting")

	go func() {
		defer st.stopWG.Done()
		for {
			select {
			case <-st.stopContext.Done():
				return
			case order := <-st.LiveOrders:
				log.Infoln("Get order ", order)
			}
		}
	}()

	log.Infoln("Started")
}

func (st *StupidStrategy) Finalize() {
	log.Infoln("Stopping")
	st.stopCancel()
	st.stopWG.Wait()
	log.Infoln("stopped")
}
