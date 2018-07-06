package strategies

import (
	"context"
	"sync"

	"github.com/yarencheng/crypto-trade/go/data"
	"github.com/yarencheng/crypto-trade/go/logger"
)

var log = logger.Get("stupid_strategy.go")

type StupidStrategy struct {
	stop       chan int
	wg         sync.WaitGroup
	LiveOrders <-chan data.Order
}

func New() *StupidStrategy {
	return &StupidStrategy{
		stop: make(chan int, 1),
	}
}

func (st *StupidStrategy) Start() {
	log.Infoln("Starting")

	st.wg.Add(1)
	go func() {
		defer st.wg.Done()
		for {
			select {
			case <-st.stop:
				return
			case order := <-st.LiveOrders:
				log.Infoln("Get order ", order)
			}
		}
	}()

	log.Infoln("Started")
}

func (st *StupidStrategy) Stop(ctx context.Context) error {
	log.Infoln("Stopping")

	wg := sync.WaitGroup{}
	wait := make(chan int)

	wg.Add(1)
	go func() {
		close(st.stop)
		st.wg.Wait()
		close(wait)
	}()

	select {
	case <-ctx.Done():
	case <-wait:
		log.Infoln("Stopped")
	}

	return ctx.Err()
}
