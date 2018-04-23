package exchanges

import (
	"time"

	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/data/currencies"
)

type DummyExchange struct {
}

func NewDummyExchange() *DummyExchange {
	var ex DummyExchange
	return &ex
}

func (ex *DummyExchange) GetName() string {
	return "Dummy Exchange"
}

func (ex *DummyExchange) Ping() float64 {
	return 0
}

func (ex *DummyExchange) GetOrders(from currencies.Currency, to currencies.Currency) (chan data.Order, error) {

	c := make(chan data.Order)

	go func() {
		for {
			c <- data.Order{
				From: currencies.BTC,
				To:   currencies.ETH,
			}
			time.Sleep(time.Second)
		}
	}()

	return c, nil
}
