package dummy

import (
	"context"
	"sync"
	"time"

	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

var log = logger.Get("dummy_exchange.go")

type Dummy struct {
	stop       chan int
	wg         sync.WaitGroup
	DelayMs    int64
	LiveOrders chan<- entity.OrderBook
}

func New() *Dummy {
	return &Dummy{
		stop:    make(chan int, 1),
		DelayMs: 1000,
	}
}

func (dummy *Dummy) Start() {
	log.Infoln("Starting")

	dummy.wg.Add(1)
	go func() {
		defer dummy.wg.Done()
		for {
			order := entity.OrderBook{
				From: entity.BTC,
				To:   entity.ETH,
			}
			select {
			case <-dummy.stop:
				return
			case dummy.LiveOrders <- order:
				log.Infoln("Create new order", order)
				time.Sleep(time.Duration(dummy.DelayMs) * time.Millisecond)
			}
		}
	}()
}

func (dummy *Dummy) Stop(ctx context.Context) error {
	log.Infoln("Stopping")

	wg := sync.WaitGroup{}
	wait := make(chan int)

	wg.Add(1)
	go func() {
		close(dummy.stop)
		dummy.wg.Wait()
		close(wait)
	}()

	select {
	case <-ctx.Done():
	case <-wait:
		log.Infoln("Stopped")
	}

	return ctx.Err()
}
