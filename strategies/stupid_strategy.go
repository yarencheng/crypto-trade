package strategies

import (
	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/logger"
)

var log = logger.Get("stupid_strategy.go")

type StupidStrategy struct {
	orders chan data.Order
}

func NewStupidStrategy() *StupidStrategy {
	return &StupidStrategy{
		orders: make(chan data.Order, 10),
	}
}

func (s *StupidStrategy) In(orders <-chan data.Order) error {
	go func() {
		for order := range orders {
			log.Infoln("Get order aaaa", order)
			s.orders <- order
		}
	}()
	return nil
}

func (s *StupidStrategy) Out() (<-chan data.Order, error) {
	return s.orders, nil
}
