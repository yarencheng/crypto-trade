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
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Bitfinex struct {
	Currencies      []entity.Currency
	OrderBooks      chan<- entity.OrderBookEvent
	stop            chan int
	stopWg          sync.WaitGroup
	subBookSymbols  *hashset.Set
	wsClient        *websocket.Conn
	channelHandlers map[int64]func(gj *gjson.Result) error
}

func New() *Bitfinex {
	b := &Bitfinex{
		stop:           make(chan int, 1),
		subBookSymbols: hashset.New(),
	}
	return b
}

func (b *Bitfinex) Start() error {
	logger.Info("Starting")
	defer logger.Info("Started")

	for _, c1 := range b.Currencies {
		for _, c2 := range b.Currencies {
			pair := hashset.New()
			pair.Add(c1, c2)
			symbol, ok := pairSymbol(pair)
			if !ok {
				continue
			}
			b.subBookSymbols.Add(symbol)
		}
	}

	b.stopWg.Add(1)
	go func() {
		defer b.stopWg.Done()

		for {
			logger.Infof("Listening to web socket")

			err := b.listenWebSocket()

			if err != nil {
				logger.Warnf("Got an error; not restart web socket. err: [%v]", err)
				return
			}

			select {
			case <-b.stop:
				return
			default:
				logger.Infof("Restart web socket")
			}
		}

	}()

	return nil
}

func (b *Bitfinex) Stop(ctx context.Context) error {
	logger.Info("Stopping")
	defer logger.Info("Stopped")

	wait := make(chan int, 1)

	go func() {
		close(b.stop)
		b.stopWg.Wait()
		wait <- 1
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
		return nil
	}
}

func pairSymbol(pair sets.Set) (string, bool) {

	if pair.Contains(entity.BTC, entity.ETH) {
		return "ETHBTC", true
	} else {
		return "", false
	}
}

func (b *Bitfinex) listenWebSocket() error {

	url := url.URL{
		Scheme: "wss",
		Host:   "api.bitfinex.com",
		Path:   "ws/2",
	}

	logger.Infof("Connecting to [%v]", url.String())

	client, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		err = fmt.Errorf("Dial to [%v] failed. err: [%v]", url, err)
		logger.Warnf("%v", err)
		return err
	}
	b.wsClient = client

	b.channelHandlers = make(map[int64]func(gj *gjson.Result) error)

	b.onWebSocketConnected()

	logger.Infof("Connected to [%v]", url.String())

	for _, symbol := range b.subBookSymbols.Values() {
		logger.Infof("Subscribe book channel with symbol [%v]", symbol)

		err = client.WriteJSON(map[string]interface{}{
			"event":   "subscribe",
			"channel": "book",
			"pair":    symbol.(string),
			"prec":    "P0",
			"freq":    "F0",
		})

		if err != nil {
			err = fmt.Errorf("Subscribe channel [%v] failed. err: [%v]", symbol, err)
			logger.Warnf("%v", err)
			return err
		}
	}

	stopLoop := make(chan int, 1)

	go func() {
		defer client.Close()
		select {
		case <-b.stop:
		case <-stopLoop:
		}

		err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Warnf("Close web socket client failed. err: [%v]", err)
		}

	}()

	for {
		// message := make(map[string]interface{}, 0)
		var message interface{}

		err = client.ReadJSON(&message)

		if err != nil {
			err = fmt.Errorf("Read message failed. err: [%v]", err)
			logger.Warnf("%v", err)
			close(stopLoop)
			return err
		}

		logger.Debugf("message [%v]", message)

		j, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("Failed to marshal json [%v]. err: [%v]", j, err)
		}

		gj := gjson.ParseBytes(j)
		logger.Debugf("gj [%v]", gj)

		if err := b.handleWsResponse(&gj); err != nil {
			return fmt.Errorf("Process message [%v] failed. err: [%v]", message, err)
		}
	}

	return nil
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

func (b *Bitfinex) handleWsResponseObject(gj *gjson.Result) error {

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

		b.channelHandlers[chanId] = func(gj *gjson.Result) error {
			if e := gj.Get("1"); e.IsArray() {
				return b.handleOrderBookSnapshot(pair, gj)
			} else {
				return b.handleOrderBookUpdate(pair, gj)
			}

		}
	default:
		return fmt.Errorf("Unknown event: [%v]", event)
	}

	return nil
}

func (b *Bitfinex) handleWsResponseArray(gj *gjson.Result) error {

	var chanID int64
	if e := gj.Get("0"); e.Exists() {
		chanID = e.Int()
	} else {
		return fmt.Errorf("array is empty")
	}

	if fn, ok := b.channelHandlers[chanID]; ok {
		return fn(gj)
	} else {
		return fmt.Errorf("No handler for channel [%v]", chanID)
	}
}

func (b *Bitfinex) handleOrderBookSnapshot(pair string, gj *gjson.Result) error {

	return nil
}

func (b *Bitfinex) handleOrderBookUpdate(pair string, gj *gjson.Result) error {

	return nil
}

func (b *Bitfinex) onWebSocketConnected() {
	b.OrderBooks <- entity.OrderBookEvent{
		Exchange: entity.Poloniex,
		Date:     time.Now(),
		Type:     entity.ExchangeRestart,
	}
}
