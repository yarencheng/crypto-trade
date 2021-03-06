package bitfinex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/tidwall/gjson"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/exchange/websocketproxy"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Bitfinex struct {
	Currencies         []entity.Currency
	OrderBooks         chan<- entity.OrderBookEvent
	stop               chan int
	stopWg             sync.WaitGroup
	subBookSymbols     *hashset.Set
	ws                 *websocketproxy.WebSocketProxy
	channelBookSymbols map[int64]string
	wsIn               <-chan *gjson.Result
	wsOut              chan<- *gjson.Result
	readStop           chan int
	readStopWg         sync.WaitGroup
}

func New() *Bitfinex {
	b := &Bitfinex{
		stop:           make(chan int, 1),
		subBookSymbols: hashset.New(),
	}

	b.ws = websocketproxy.New(&websocketproxy.Config{
		URL: url.URL{
			Scheme: "wss",
			Host:   "api.bitfinex.com",
			Path:   "ws/2",
		},
	})

	return b
}

func (this *Bitfinex) Start() error {
	logger.Info("Starting")

	for _, c1 := range this.Currencies {
		for _, c2 := range this.Currencies {
			pair := hashset.New()
			pair.Add(c1, c2)
			symbol, ok := pairSymbol(pair)
			if !ok {
				continue
			}
			this.subBookSymbols.Add(symbol)
		}
	}

	this.ws.SetConnectedHandler(this.OnWsConnected)
	this.ws.SetDisconnectedHandler(this.OnDisconnected)
	this.ws.SetPingTooLongFnHandler(func(delay time.Duration) {
		logger.Warnf("Restart server since ping need [%v] seconds", delay.Seconds())
		// restart
		go func() {
			logger.Info()
			if err := this.ws.Disconnect(); err != nil {
				logger.Warnf("Disconnect from server failed. err:[%v]", err)
			}

			for {
				if err := this.ws.Connect(); err != nil {
					logger.Warnf("Restart websocket failed since [%v]. Sleep 10 seconds and then try again.", err)
					time.Sleep(10 * time.Second)
				} else {
					break
				}
			}
		}()
	})

	err := this.ws.Connect()
	if err != nil {
		err = fmt.Errorf("Start websocket failed. err: [%v]", err)
		logger.Errorf("%v", err)
		return err
	}

	logger.Info("Started")

	return nil
}

func (p *Bitfinex) Stop(ctx context.Context) error {
	logger.Info("Stopping")

	wait := make(chan int, 1)

	p.stopWg.Add(1)
	go func() {
		defer p.stopWg.Done()
		err := p.ws.Stop(ctx)
		if err != nil {
			logger.Errorf("Stop web socket failed. err: [%v]", err)
		}
	}()

	go func() {
		close(p.stop)
		p.stopWg.Wait()
		wait <- 1
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
		logger.Info("Stopped")
		return nil
	}
}

func (this *Bitfinex) OnWsConnected(in <-chan *gjson.Result, out chan<- *gjson.Result) {
	logger.Infof("Web socket connected")
	logger.Infof("Reader is starting")

	this.wsIn = in
	this.wsOut = out

	this.readStop = make(chan int, 1)

	this.OrderBooks <- entity.OrderBookEvent{
		OrderBook: entity.OrderBook{
			Exchange: entity.Bitfinex,
			Date:     time.Now(),
		},
		Type: entity.ExchangeRestart,
	}

	for _, symbol := range this.subBookSymbols.Values() {
		logger.Infof("Subscribe book channel with symbol [%v]", symbol)

		jb, err := json.Marshal(map[string]interface{}{
			"event":   "subscribe",
			"channel": "book",
			"pair":    symbol.(string),
			"prec":    "P0",
			"freq":    "F0",
		})

		if err != nil {
			logger.Errorf("Create JSON failed. err: [%v]", err)
			continue
		}

		gj := gjson.ParseBytes(jb)
		this.wsOut <- &gj
	}

	this.readStopWg.Add(1)
	go func() {
		defer this.readStopWg.Done()

		this.channelBookSymbols = make(map[int64]string)
		defer func() { this.channelBookSymbols = nil }()

		for {
			select {
			case <-this.readStop:
				logger.Infof("Reader exists.")
			case gj, ok := <-this.wsIn:
				if ok {
					logger.Debugf("Read: %v", gj.String())
					if err := this.handleWsResponse(gj); err != nil {
						logger.Errorf("Handle responce failed. err: [%v]", err)
						return
					}
				} else {
					logger.Infof("Input channel is closed. Reader exists.")
					return
				}
			}
		}
	}()

	logger.Infof("Reader is started")
}

func (this *Bitfinex) OnDisconnected() {
	logger.Infof("Web socket disconnected.")

	close(this.readStop)
	this.readStopWg.Wait()

	this.wsOut = nil
	this.wsIn = nil
}

func pairSymbol(pair sets.Set) (string, bool) {

	if pair.Contains(entity.BTC, entity.ETH) {
		return "ETHBTC", true
	} else {
		return "", false
	}
}

func (b *Bitfinex) handleWsResponse(gj *gjson.Result) error {

	if gj.IsObject() {
		return b.handleWsResponseObject(gj)
	} else if gj.IsArray() {
		return b.handleWsResponseArray(gj)
	} else {
		return fmt.Errorf("Should be a json object or array")
	}
}

func (this *Bitfinex) handleWsResponseObject(gj *gjson.Result) error {

	var event string
	if e := gj.Get("event"); !e.Exists() {
		return fmt.Errorf("missing element 'event'")
	} else {
		event = e.String()
	}
	logger.Debugf("event = [%v]", event)

	switch event {
	case "info":
		logger.Infof("Bitfin wws info: [%v]", gj.String())
		return nil
	case "subscribed":

		var chanId int64
		if e := gj.Get("chanId"); e.Exists() {
			chanId = e.Int()
		} else {
			return fmt.Errorf("chanId is missing")
		}
		logger.Debugf("chanId: [%v]", chanId)

		var pair string
		if e := gj.Get("pair"); e.Exists() {
			pair = e.String()
		} else {
			return fmt.Errorf("pair is missing")
		}
		logger.Debugf("pair: [%v]", pair)

		logger.Infof("subscribed pair [%v] at channel [%v]", pair, chanId)

		this.channelBookSymbols[chanId] = pair
	default:
		return fmt.Errorf("Unknown event: [%v]", event)
	}

	return nil
}

func (this *Bitfinex) handleWsResponseArray(gj *gjson.Result) error {

	var chanID int64
	if e := gj.Get("0"); e.Exists() {
		chanID = e.Int()
	} else {
		return fmt.Errorf("array is empty")
	}

	if symbol, ok := this.channelBookSymbols[chanID]; ok {
		return this.handleOrderBook(symbol, gj)
	} else {
		return fmt.Errorf("No symbol for channel [%v]", chanID)
	}
}

func (this *Bitfinex) handleOrderBook(pair string, gj *gjson.Result) error {

	data := gj.Get("1")
	if data.IsArray() {

	} else if data.String() == "hb" {
		logger.Debugf("[%v] heart beat", pair)
		return nil
	} else {
		return fmt.Errorf("Unknown data [%v]", data.String())
	}

	if data.Get("0").IsArray() {
		for _, e := range data.Array() {
			if err := this.handleOrderBookUpdate(pair, &e); err != nil {
				return err
			}
		}
	} else {
		if err := this.handleOrderBookUpdate(pair, &data); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bitfinex) handleOrderBookUpdate(pair string, gj *gjson.Result) error {

	c1, c2, err := getCurrencies(pair)
	if err != nil {
		return err
	}

	price := gj.Get("0").Float()
	count := gj.Get("1").Float()
	amount := gj.Get("2").Float()

	var order *entity.OrderBookEvent
	if amount > 0 {
		order = &entity.OrderBookEvent{
			Type: entity.Update,
			OrderBook: entity.OrderBook{
				Exchange: entity.Bitfinex,
				Date:     time.Now(),
				From:     c2,
				To:       c1,
				Price:    price,
				Volume:   amount * count,
			},
		}
	} else {
		order = &entity.OrderBookEvent{
			Type: entity.Update,
			OrderBook: entity.OrderBook{
				Exchange: entity.Bitfinex,
				Date:     time.Now(),
				From:     c1,
				To:       c2,
				Price:    1 / price,
				Volume:   -amount * count,
			},
		}
	}

	logger.Debugf("Receive order [%#v]", order)

	b.OrderBooks <- *order

	return nil
}

func getCurrencies(symbol string) (entity.Currency, entity.Currency, error) {
	switch symbol {
	case "ETHBTC":
		return entity.ETH, entity.BTC, nil
	default:
		return "", "", fmt.Errorf("Unknown symbol [%v]", symbol)
	}
}
