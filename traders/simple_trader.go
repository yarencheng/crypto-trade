package traders

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/exchanges"
	"github.com/yarencheng/crypto-trade/tasks"
)

type SimpleTrader struct {
	Exchanges  []exchanges.ExchangeI
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

func (trader *SimpleTrader) AddExchange(exchange exchanges.ExchangeI) {
	trader.Exchanges = append(trader.Exchanges, exchange)
}

func (trader *SimpleTrader) GetStatus() tasks.Status {
	trader.statusLock.Lock()
	defer trader.statusLock.Unlock()
	return trader.status
}

func (trader *SimpleTrader) Start() error {
	logrus.Errorln("aaa 1")
	trader.statusLock.Lock()
	if trader.status != tasks.Pending {
		s := fmt.Sprint(trader, "is not pending")
		logrus.Warnln(s)
		logrus.Errorln("aaa 2")
		return errors.New(s)
	}
	trader.status = tasks.Running
	trader.statusLock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	for _, ex := range trader.Exchanges {

		go func() {
			orders, _ := ex.GetOrders(data.BTC, data.ETH)
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
	logrus.Errorln("aaa 3")
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
