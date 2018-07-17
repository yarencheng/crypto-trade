package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/yarencheng/crypto-trade/go/exchange/websocketproxy"

	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Poloniex struct {
	Currencies  []entity.Currency
	OrderBooks  chan<- entity.OrderBookEvent
	stop        chan int
	stopWg      sync.WaitGroup
	wsSeqenceID map[int64]int64 // key: channel_id, value sequence
	subChannels []string
	ws          *websocketproxy.WebSocketProxy
	wsIn        <-chan *gjson.Result
	wsOut       chan<- *gjson.Result
	readStop    chan int
	readStopWg  sync.WaitGroup
}

func New() *Poloniex {
	b := &Poloniex{
		stop:        make(chan int, 1),
		subChannels: make([]string, 0),
	}

	b.ws = websocketproxy.New(&websocketproxy.Config{
		URL: url.URL{
			Scheme: "wss",
			Host:   "api2.poloniex.com",
		},
	})

	return b
}

func (this *Poloniex) Start() error {
	logger.Info("Starting")

	ids := hashset.New()
	for _, c1 := range this.Currencies {
		for _, c2 := range this.Currencies {
			pair := hashset.New()
			pair.Add(c1, c2)
			id, ok := pairToChannelID(pair)
			if !ok {
				continue
			}
			ids.Add(id)
		}
	}

	for _, id := range ids.Values() {
		this.subChannels = append(this.subChannels, id.(string))
	}

	this.ws.SetConnectedHandler(this.OnWsConnected)
	this.ws.SetDisconnectedHandler(this.OnDisconnected)
	this.ws.SetPingTooLongFnHandler(func(delay time.Duration) {
		logger.Error("TODO Ping error")
		go func() {
			this.ws.Disconnect()
			for {
				logger.Error("TODO reconnect")
				if nil == this.ws.Connect() {
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

func (p *Poloniex) Stop(ctx context.Context) error {
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

func (this *Poloniex) OnWsConnected(in <-chan *gjson.Result, out chan<- *gjson.Result) {
	logger.Infof("Web socket connected")
	logger.Infof("Reader is starting")

	this.wsIn = in
	this.wsOut = out

	this.readStop = make(chan int, 1)

	this.readStopWg.Add(1)
	go func() {
		defer this.readStopWg.Done()

		for {
			select {
			case <-this.readStop:
				logger.Infof("Reader exists.")
			case gj, ok := <-this.wsIn:
				if ok {
					logger.Infof("Read: %v", gj.String())
				} else {
					logger.Infof("Input channel is closed. Reader exists.")
					return
				}
			}
		}
	}()

	logger.Infof("Reader is started")
}

func (this *Poloniex) OnDisconnected() {
	logger.Infof("Web socket disconnected.")

	close(this.readStop)
	this.readStopWg.Wait()

	this.wsOut = nil
	this.wsIn = nil
}

// old ==================

func (p *Poloniex) OnWsConnects() {
	logger.Infof("Web socket connected")
	p.OrderBooks <- entity.OrderBookEvent{
		Exchange: entity.Poloniex,
		Date:     time.Now(),
		Type:     entity.ExchangeRestart,
	}
}

func (p *Poloniex) listenWebSocket() error {

	url := url.URL{
		Scheme: "wss",
		Host:   "api2.poloniex.com",
	}

	logger.Infof("Connecting to [%v]", url.String())

	client, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		err = fmt.Errorf("Dial to [%v] failed. err: [%v]", url, err)
		logger.Warnf("%v", err)
		return err
	}

	logger.Infof("Connected to [%v]", url.String())

	p.wsSeqenceID = make(map[int64]int64)

	logger.Infof("Subscribe heart beat")

	for _, id := range p.subChannels {
		logger.Infof("Subscribe channel [%v]", id)
		j := "{\"command\":\"subscribe\",\"channel\": \"" + id + "\"}"
		err = client.WriteMessage(websocket.TextMessage, []byte(j))
		if err != nil {
			err = fmt.Errorf("Subscribe channel [%v] failed. err: [%v]", id, err)
			logger.Warnf("%v", err)
			return err
		}
	}

	stopLoop := make(chan int, 1)

	go func() {
		defer client.Close()
		select {
		case <-p.stop:
		case <-stopLoop:
		}

		err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			logger.Warnf("Close web socket client failed. err: [%v]", err)
		}

		p.wsSeqenceID = nil
	}()

	for {
		message := make([]interface{}, 0)
		err = client.ReadJSON(&message)

		if err != nil {
			err = fmt.Errorf("Read message failed. err: [%v]", err)
			logger.Warnf("%v", err)
			close(stopLoop)
			return err
		}

		j, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("Failed to marshal json [%v]. err: [%v]", j, err)
		}

		gj := gjson.ParseBytes(j)

		logger.Debugf("message [%v]", gj)

		if err := p.handleWebsockerResponse(&gj); err != nil {
			return fmt.Errorf("Process message [%v] failed. err: [%v]", message, err)
		}
	}

	return nil
}

func (p *Poloniex) handleWebsockerResponse(gj *gjson.Result) error {

	if !gj.IsArray() {
		return fmt.Errorf("Not a json array")
	}

	ar := gj.Array()

	var ch int64
	if len(ar) <= 0 {
		return fmt.Errorf("1st element 'channel_id' is missing")
	} else {
		ch = ar[0].Int()
	}

	if ch == 1010 {
		logger.Debugf("Receive heart beat")
		return nil
	}

	var sequence int64
	if len(ar) <= 1 {
		return fmt.Errorf("2nd element 'sequence_number' is missing")
	} else {
		sequence = ar[1].Int()
	}

	if old, ok := p.wsSeqenceID[ch]; ok {
		if old >= sequence {
			return fmt.Errorf("Channel [%v] receive new sequence [%v] is small then last one [%v]", ch, sequence, old)
		}
	}

	p.wsSeqenceID[ch] = sequence

	if !ar[2].IsArray() {
		return fmt.Errorf("Not array")
	}

	ar = ar[2].Array()

	var err error
	for i := 0; i < len(ar); i++ {
		switch ch {
		case 148: // BTC_ETH
			err = p.handlePriceAggregatedBook(entity.BTC, entity.ETH, &ar[i])
		default:
			return fmt.Errorf("Unknown channel ID [%v]", ch)
		}
	}

	return err
}

func (p *Poloniex) handlePriceAggregatedBook(c1, c2 entity.Currency, gj *gjson.Result) error {

	if !gj.IsArray() {
		return fmt.Errorf("Not a json array")
	}

	ar := gj.Array()

	if len(ar) <= 0 {
		return fmt.Errorf("op code is missing")
	}

	op := ar[0].String()

	switch op {
	case "i":
		if len(ar) != 2 {
			return fmt.Errorf("need 2 element")
		}

		asks := ar[1].Get("orderBook.0")
		bids := ar[1].Get("orderBook.1")

		if (!asks.Exists()) || (!asks.IsObject()) {
			return fmt.Errorf("asks is missing")
		}

		if (!bids.Exists()) || (!bids.IsObject()) {
			return fmt.Errorf("bids is missing")
		}

		for ps, vs := range asks.Map() {

			price, err := strconv.ParseFloat(ps, 64)
			if err != nil {
				return fmt.Errorf("[%v]is not float", ps)
			}

			volume := vs.Float()

			order := entity.OrderBookEvent{
				Type:     entity.Update,
				Exchange: entity.Poloniex,
				Date:     time.Now(),
				From:     c2,
				To:       c1,
				Price:    1 / price,
				Volume:   price * volume,
			}

			logger.Debugf("Receive order [%v]", order)

			p.OrderBooks <- order
		}

		for ps, vs := range bids.Map() {

			price, err := strconv.ParseFloat(ps, 64)
			if err != nil {
				return fmt.Errorf("[%v]is not float", ps)
			}

			volume := vs.Float()

			order := entity.OrderBookEvent{
				Type:     entity.Update,
				Exchange: entity.Poloniex,
				Date:     time.Now(),
				From:     c1,
				To:       c2,
				Price:    price,
				Volume:   volume,
			}

			logger.Debugf("Receive order [%v]", order)

			p.OrderBooks <- order
		}

	case "o":

		if len(ar) != 4 {
			return fmt.Errorf("need 4 element")
		}

		isAsk := ar[1].Int() == 0
		price := ar[2].Float()
		volume := ar[3].Float()

		var order *entity.OrderBookEvent
		if isAsk {
			order = &entity.OrderBookEvent{
				Type:     entity.Update,
				Exchange: entity.Poloniex,
				Date:     time.Now(),
				From:     c2,
				To:       c1,
				Price:    1 / price,
				Volume:   price * volume,
			}
		} else {
			order = &entity.OrderBookEvent{
				Type:     entity.Update,
				Exchange: entity.Poloniex,
				Date:     time.Now(),
				From:     c1,
				To:       c2,
				Price:    price,
				Volume:   volume,
			}
		}

		logger.Debugf("Receive order [%v]", order)

		p.OrderBooks <- *order
	}

	return nil
}

func pairToChannelID(pair sets.Set) (string, bool) {

	if pair.Contains(entity.BTC, entity.ETH) {
		return "148", true
	} else {
		return "", false
	}
}
