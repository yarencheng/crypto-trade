package strategies

import (
	"context"
	"sync"

	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type StupidStrategy struct {
	stop      chan int
	wg        sync.WaitGroup
	InOrders  <-chan entity.OrderBookEvent
	OutOrders chan<- entity.BuyOrderEvent
}

func NewStupidStrategy() *StupidStrategy {
	return &StupidStrategy{
		stop: make(chan int, 1),
	}
}

func (this *StupidStrategy) Start() error {
	logger.Infoln("Starting")

	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		this.worker()
	}()

	logger.Infoln("Started")

	return nil
}

func (this *StupidStrategy) Stop(ctx context.Context) error {
	logger.Infoln("Stopping")

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
		logger.Infoln("Stopped")
	}

	return ctx.Err()
}

func (this *StupidStrategy) worker() {
	logger.Infoln("Worker started")
	defer logger.Infoln("Worker finished")

	for {
		select {
		case <-this.stop:
			break
		case <-this.InOrders:
			this.OutOrders <- entity.BuyOrderEvent{}
		}
	}

}
