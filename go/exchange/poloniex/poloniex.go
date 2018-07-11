package poloniex

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"time"

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
			if err := p.processChannelResponce(message); err != nil {
				err = fmt.Errorf("Process message failed. err: [%v]", err)
				logger.Warnf("%v", err)
				return
			}
		}
	}()

	p.stopWg.Add(1)
	go func() {
		defer p.stopWg.Done()
		defer c.Close()

		<-p.stop

		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Warnf("Close web socket client failed. err: [%v]", err)
		}
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

func (p *Poloniex) processChannelResponce(r []interface{}) error {

	logger.Debugf("channel responce: [%#v]", r)

	var channelID int
	var sequenceNumber int
	var err error

	switch r[0].(type) {
	case string:
		channelID, err = strconv.Atoi(r[0].(string))
		if err != nil {
			return fmt.Errorf("Parse string to int failed. err: [%v]", err)
		}
	case float64:
		channelID = int(r[0].(float64))
	default:
		return fmt.Errorf("The 1st element should be a float64 or string but a [%v]", reflect.TypeOf(r[0]))
	}

	if channelID == 1010 {
		//heart beat
		return nil
	}

	switch r[1].(type) {
	case float64:
		sequenceNumber = int(r[1].(float64))
	default:
		return fmt.Errorf("The 2nd element should be a float64 but a [%v]", reflect.TypeOf(r[1]))
	}

	logger.Debugf("channelID = [%v], sequenceNumber = [%v]", channelID, sequenceNumber)

	for index := 2; index < len(r); index++ {

		payload1, ok := r[index].([]interface{})
		if !ok {
			return fmt.Errorf("The [%v] element should be a []interface{} but a [%v]", index, reflect.TypeOf(r[index]))
		}

		payload2, ok := payload1[0].([]interface{})
		if !ok {
			return fmt.Errorf("The [%v] element should be a []interface{} but a [%v]", index, reflect.TypeOf(payload1[0]))
		}

		switch channelID {
		case 148:
			if err := p.processAggregatedBookPayload(entity.BTC, entity.ETH, payload2); err != nil {
				return fmt.Errorf("Process payload failed. err: [%v]", err)
			}
		default:
			return fmt.Errorf("Unsupported channel [%v]", channelID)
		}
	}

	return nil
}

func (p *Poloniex) processAggregatedBookPayload(from, to entity.Currency, payload []interface{}) error {

	logger.Debugf("from: [%v]", from)
	logger.Debugf("to: [%v]", to)
	logger.Debugf("payload: [%v]", payload)

	op, ok := payload[0].(string)
	if !ok {
		return fmt.Errorf("The 1st element is not a string but a [%v]", reflect.TypeOf(payload[0]))
	}

	switch op {
	case "o":
		isAsk := payload[1].(float64) == 0

		price, err := strconv.ParseFloat(payload[2].(string), 64)
		if err != nil {
			return fmt.Errorf("[%v] is not a float64, err: [%v]", payload[2], err)
		}

		volume, err := strconv.ParseFloat(payload[3].(string), 64)
		if err != nil {
			return fmt.Errorf("[%v] is not a float64, err: [%v]", payload[3], err)
		}

		if isAsk {
			p.OrderBooks <- entity.OrderBook{
				Exchange: entity.Poloniex,
				Time:     time.Now(),
				From:     entity.ETH,
				To:       entity.BTC,
				Price:    1 / price,
				Volume:   price * volume,
			}
		} else {
			p.OrderBooks <- entity.OrderBook{
				Exchange: entity.Poloniex,
				Time:     time.Now(),
				From:     entity.BTC,
				To:       entity.ETH,
				Price:    price,
				Volume:   volume,
			}
		}

	case "t":
		logger.Warnf("TODO: %#v", payload)
	case "i":
		m, ok := payload[1].(map[string]interface{})
		if !ok {
			return fmt.Errorf("The 2nd element is not a map[string]interface{} but a [%v]", reflect.TypeOf(payload[1]))
		}

		orderBook, ok := m["orderBook"].([]interface{})
		if !ok {
			return fmt.Errorf("The \"orderBook\" element is not a []interface{} but a [%v]", reflect.TypeOf(m["orderBook"]))
		}

		asks, ok := orderBook[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("The 1st element of \"orderBook\" is not a map[string]interface{} but a [%v]", reflect.TypeOf(orderBook[0]))
		}

		bids, ok := orderBook[1].(map[string]interface{})
		if !ok {
			return fmt.Errorf("The 2nd element of \"orderBook\" is not a map[string]interface{} but a [%v]", reflect.TypeOf(orderBook[1]))
		}

		for price, volume := range asks {

			pf, err := strconv.ParseFloat(price, 64)
			if err != nil {
				return fmt.Errorf("[%v] is not a float64, err: [%v]", price, err)
			}

			vf, err := strconv.ParseFloat(volume.(string), 64)
			if err != nil {
				return fmt.Errorf("[%v] is not a float64, err: [%v]", vf, err)
			}

			p.OrderBooks <- entity.OrderBook{
				Exchange: entity.Poloniex,
				Time:     time.Now(),
				From:     entity.ETH,
				To:       entity.BTC,
				Price:    1 / pf,
				Volume:   pf * vf,
			}
		}

		for price, volume := range bids {
			pf, err := strconv.ParseFloat(price, 64)
			if err != nil {
				return fmt.Errorf("[%v] is not a float64, err: [%v]", price, err)
			}

			vf, err := strconv.ParseFloat(volume.(string), 64)
			if err != nil {
				return fmt.Errorf("[%v] is not a float64, err: [%v]", vf, err)
			}

			p.OrderBooks <- entity.OrderBook{
				Exchange: entity.Poloniex,
				Time:     time.Now(),
				From:     entity.BTC,
				To:       entity.ETH,
				Price:    pf,
				Volume:   vf,
			}
		}

	default:
		return fmt.Errorf("unsupported op [%v]", op)
	}

	return nil
}
