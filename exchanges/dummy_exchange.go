package exchanges

import (
	"container/list"
	"sync"
	"time"

	"github.com/yarencheng/crypto-trade/data"
)

type DummyExchange struct {
	outOrders      *list.List
	outOrdersMutex sync.Mutex
}

func NewDummyExchange() *DummyExchange {
	return &DummyExchange{
		outOrders: list.New(),
	}
}

func (ex *DummyExchange) GetName() string {
	return "Dummy Exchange"
}

func (ex *DummyExchange) Ping() float64 {
	return 0
}

func (ex *DummyExchange) Finalize() {
	ex.outOrdersMutex.Lock()
	defer ex.outOrdersMutex.Unlock()

	c := ex.outOrders.Front()
	for c != nil {
		c.Value.(chan int) <- 0
		c = c.Next()
	}
}

func (ex *DummyExchange) GetOrders(from data.Currency, to data.Currency) (<-chan data.Order, error) {

	c := make(chan data.Order, 10)
	stop := make(chan int)

	ex.outOrdersMutex.Lock()
	ex.outOrders.PushBack(stop)
	ex.outOrdersMutex.Unlock()

	go func() {
		for {
			select {
			case c <- data.Order{
				From: data.BTC,
				To:   data.ETH,
			}:
				time.Sleep(time.Second)
			case <-stop:
				break
			}
		}
	}()

	return c, nil
}

func (ex *DummyExchange) Buy(from data.Order) chan data.TradeResult {
	c := make(chan data.TradeResult)
	c <- data.TradeResult{IsSucess: true}
	return c
}
