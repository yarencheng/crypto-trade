package strategies

import (
	"context"
	"sync"

	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

var log = logger.Get("stupid_strategy.go")

type StupidStrategy struct {
	stop       chan int
	wg         sync.WaitGroup
	LiveOrders <-chan entity.OrderBook
}

func New() *StupidStrategy {
	return &StupidStrategy{
		stop: make(chan int, 1),
	}
}

func (this *StupidStrategy) Start() error {
	log.Infoln("Starting")

	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		this.worker()
	}()

	log.Infoln("Started")

	return nil
}

func (this *StupidStrategy) Stop(ctx context.Context) error {
	log.Infoln("Stopping")

	wg := sync.WaitGroup{}
	wait := make(chan int)

	wg.Add(1)
	go func() {
		close(this.stop)
		this.wg.Wait()
		close(wait)
	}()

	select {
	case <-ctx.Done():
	case <-wait:
		log.Infoln("Stopped")
	}

	return ctx.Err()
}

func (this *StupidStrategy) worker() {
	log.Infoln("Worker started")
	defer log.Infoln("Worker finished")

}
