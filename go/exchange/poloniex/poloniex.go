package poloniex

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/gorilla/websocket"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Poloniex struct {
	Currencies []entity.Currency
	OrderBooks chan<- entity.OrderBook
	stop       chan int
	stopWg     sync.WaitGroup
}

func New() *Poloniex {
	b := &Poloniex{
		stop: make(chan int, 1),
	}
	return b
}

func (p *Poloniex) Start() error {
	logger.Info("Starting")
	defer logger.Info("Started")

	err := p.startAggregatedBook()
	if err != nil {
		err = fmt.Errorf("Listen order book failed. err: [%v]", err)
		logger.Warnf("%v", err)
		return err
	}

	return nil
}

func (p *Poloniex) Stop(ctx context.Context) error {
	logger.Info("Stopping")
	defer logger.Info("Stopped")

	wait := make(chan int, 1)

	go func() {
		close(p.stop)
		p.stopWg.Wait()
		wait <- 1
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
		return nil
	}
}

func (p *Poloniex) startAggregatedBook() error {

	u := url.URL{
		Scheme: "wss",
		Host:   "api2.poloniex.com",
	}

	logger.Infof("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		err = fmt.Errorf("Dial to [%v] failed. err: [%v]", u, err)
		logger.Warnf("%v", err)
		return err
	}

	for _, c1 := range p.Currencies {
		for _, c2 := range p.Currencies {

			pair := hashset.New()
			pair.Add(c1, c2)

			id, ok := pairToChannelID(pair)
			if !ok {
				continue
			}

			logger.Infof("subscribe channel [%v]", id)
			json := "{\"command\":\"subscribe\",\"channel\": \"" + id + "\"}"
			err := c.WriteMessage(websocket.TextMessage, []byte(json))
			if err != nil {
				err = fmt.Errorf("Send message [%v] failed. err: [%v]", json, err)
				logger.Warnf("%v", err)
				return err
			}
		}
	}

	p.stopWg.Add(1)
	go func() {
		defer p.stopWg.Done()
		for {
			message := make([]interface{}, 0)
			err := c.ReadJSON(&message)

			if err != nil {
				err = fmt.Errorf("Read message failed. err: [%v]", err)
				logger.Warnf("%v", err)
				return
			}
			logger.Infof("recv: %s", message)
		}
	}()

	p.stopWg.Add(1)
	go func() {
		defer p.stopWg.Done()
		defer c.Close()
		<-p.stop
	}()

	return nil
}

func pairToChannelID(pair sets.Set) (string, bool) {

	if pair.Contains(entity.BTC, entity.ETH) {
		return "148", true
	} else {
		return "", false
	}
}
