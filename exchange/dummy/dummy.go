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
	stop       chan int
	wg         sync.WaitGroup
	DelayMs    int64
	LiveOrders chan<- data.Order
}

func New() *Dummy {
	return &Dummy{
		stop:    make(chan int),
		DelayMs: 1000,
	}
}

func (dummy *Dummy) Start() {
	log.Infoln("Starting")

	dummy.wg.Add(1)
	go func() {
		defer dummy.wg.Done()
		for {
			order := data.Order{
				From: data.BTC,
				To:   data.ETH,
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
