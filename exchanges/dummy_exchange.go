package exchanges

import (
	"time"

	"github.com/yarencheng/crypto-trade/data"
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

func (ex *DummyExchange) GetOrders(from data.Currency, to data.Currency) (<-chan data.Order, error) {

	c := make(chan data.Order)

	go func() {
		for {
			c <- data.Order{
				From: data.BTC,
				To:   data.ETH,
			}
			time.Sleep(time.Second)
		}
	}()

	return c, nil
}

func (ex *DummyExchange) Buy(from data.Order) chan data.TradeResult {
	c := make(chan data.TradeResult)
	c <- data.TradeResult{IsSucess: true}
	return c
}
