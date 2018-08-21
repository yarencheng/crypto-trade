package strategies

import (
	"context"
	"fmt"
	"sync"

	"github.com/emirpasic/gods/trees/avltree"
	"github.com/emirpasic/gods/utils"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type StupidStrategy struct {
	stop      chan int
	wg        sync.WaitGroup
	InOrders  <-chan entity.OrderBookEvent
	OutOrders chan<- entity.BuyOrderEvent
	// exchange > from > to > price > volume
	priceBook map[entity.Exchange]map[entity.Currency]map[entity.Currency]*avltree.Tree
}

func NewStupidStrategy() *StupidStrategy {
	return &StupidStrategy{
		stop:      make(chan int, 1),
		priceBook: make(map[entity.Exchange]map[entity.Currency]map[entity.Currency]*avltree.Tree),
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
			return
		case order := <-this.InOrders:
			logger.Debugf("Receive an order [%v]", order)

			err := this.updatePriceBook(&order)
			if err != nil {
				logger.Warnf("Update order [%v] failed. err: [%v]", err)
				return
			}

			// for each pair in a exchange
			for ex1, _ := range this.priceBook {
				for cur1, _ := range this.priceBook[ex1] {
					for cur2, books1 := range this.priceBook[ex1][cur1] {

						// find all reverse pair in other exchanges
						for ex2, _ := range this.priceBook {
							if ex1 == ex2 {
								break
							}
							if _, exist := this.priceBook[ex2][cur2]; !exist {
								break
							}
							if _, exist := this.priceBook[ex2][cur2][cur1]; !exist {
								break
							}
							books2 := this.priceBook[ex2][cur2][cur1]

							high1, ok1 := books1.Floor(float64(9999999))
							high2, ok2 := books2.Floor(float64(9999999))

							if !ok1 || !ok2 {
								continue
							}

							v1 := high1.Key.(float64)
							v2 := high2.Key.(float64)

							// logger.Warnf("%v %v = %v", v1, v2, v1*v2)

							fee := 0.001
							if v1*v2*(1-fee)*(1-fee) <= 1 {
								continue
							}

							logger.Infof("[%v-%v-%v]=[$%v,%v], [%v-%v-%v]=[$%v,%v] %v%%",
								ex1, cur1, cur2, high1.Key, high1.Value.(float64),
								ex2, cur2, cur1, high2.Key, high2.Value.(float64),
								v1*v2*(1-fee)*(1-fee)-1,
							)

							// return

							/*

								1 * x = b
								b * y > 1   x*y>1


								1 * x * (1-z) = b
								b * y * (1-z) > 1
								(x - xz) * (y - yz) > 1
								xy + xyz^2 - 2xyz > 1
								xy (1 + z^2 - 2z) > 1
								xy (1-z)^2 > 1

							*/
						}
					}
				}
			}

			this.OutOrders <- entity.BuyOrderEvent{
				Type: entity.None,
			}
			// this.OutOrders <- entity.BuyOrderEvent{
			// 	CreateDate: time.Now(),
			// 	Type:       entity.FillOrKill,
			// 	OrderBook: entity.OrderBook{
			// 		Exchange: order.Exchange,
			// 		From:     order.From,
			// 		To:       order.To,
			// 		Price:    order.Price,
			// 		Volume:   order.Volume,
			// 	},
			// }
		}
	}
}

func (this *StupidStrategy) updatePriceBook(order *entity.OrderBookEvent) error {
	switch order.Type {
	case entity.ExchangeRestart:
		delete(this.priceBook, order.Exchange)
		return nil
	case entity.Update:
	default:
		return fmt.Errorf("Unknown even type [%v]", order.Type)
	}

	var ok bool

	var from map[entity.Currency]map[entity.Currency]*avltree.Tree
	if from, ok = this.priceBook[order.Exchange]; !ok {
		from = make(map[entity.Currency]map[entity.Currency]*avltree.Tree)
		this.priceBook[order.Exchange] = from
	}

	var to map[entity.Currency]*avltree.Tree
	if to, ok = from[order.From]; !ok {
		to = make(map[entity.Currency]*avltree.Tree)
		from[order.From] = to
	}

	var price *avltree.Tree
	if price, ok = to[order.To]; !ok {
		price = avltree.NewWith(utils.Float64Comparator)
		to[order.To] = price
	}

	if order.Volume == 0 {
		price.Remove(order.Price)
	} else {
		price.Put(order.Price, order.Volume)
	}

	return nil
}
