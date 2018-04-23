package traders

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data/currencies"
	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/tasks"
)

type SimpleTrader struct {
	exchanges  []exchanges.Exchange
	status     tasks.Status
	statusLock sync.Mutex
	stop       chan int
}

func NewSimpleTrader() *SimpleTrader {

	trader := &SimpleTrader{}

	trader.statusLock.Lock()
	trader.status = tasks.Pending
	trader.statusLock.Unlock()

	trader.stop = make(chan int)

	return trader
}

func (trader *SimpleTrader) String() string {
	return fmt.Sprintf("SimpleTrader[@%p]", trader)
}

func (trader *SimpleTrader) AddExchange(exchange exchanges.Exchange) {
	trader.exchanges = append(trader.exchanges, exchange)
}

func (trader *SimpleTrader) GetStatus() tasks.Status {
	trader.statusLock.Lock()
	defer trader.statusLock.Unlock()
	return trader.status
}

func (trader *SimpleTrader) Start() error {

	trader.statusLock.Lock()
	if trader.status != tasks.Pending {
		s := fmt.Sprint(trader, "is not pending")
		logrus.Warnln(s)
		return errors.New(s)
	}
	trader.status = tasks.Running
	trader.statusLock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	for _, ex := range trader.exchanges {

		go func() {
			orders, _ := ex.GetOrders(currencies.BTC, currencies.ETH)
			for {
				select {
				case order := <-orders:
					logrus.Infoln("Get order", order)
				case <-ctx.Done():
					break
				}
			}
		}()
	}

	go func() {
		<-trader.stop
		cancel()
		trader.statusLock.Lock()
		trader.status = tasks.Stopped
		trader.statusLock.Unlock()
	}()

	return nil
}

func (trader *SimpleTrader) Stop() error {

	trader.statusLock.Lock()
	if trader.status != tasks.Running {
		s := fmt.Sprint(trader, "is not running. status=", trader.status)
		logrus.Warnln(s)
		return errors.New(s)
	}
	trader.statusLock.Unlock()

	trader.stop <- 0

	return nil
}
