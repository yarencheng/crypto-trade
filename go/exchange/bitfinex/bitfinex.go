package bitfinex

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	bitfinex "github.com/bitfinexcom/bitfinex-api-go/v2"
	"github.com/bitfinexcom/bitfinex-api-go/v2/websocket"
	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Bitfinex struct {
	Currencies []entity.Currency
	OrderBooks chan<- entity.OrderBook
	stop       chan int
	stopWg     sync.WaitGroup
}

func New() *Bitfinex {
	b := &Bitfinex{
		stop: make(chan int, 1),
	}
	return b
}

func (b *Bitfinex) Start() error {
	logger.Info("Starting")
	defer logger.Info("Started")

	b.startListenOrderBook()

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

func (b *Bitfinex) startListenOrderBook() error {

	symbols := hashset.New()

	for _, c1 := range b.Currencies {
		for _, c2 := range b.Currencies {

			pair := hashset.New()
			pair.Add(c1, c2)

			if symbol, ok := convertToSymbol(pair); ok {
				symbols.Add(symbol)
			}
		}
	}

	client := websocket.New()
	err := client.Connect()
	if err != nil {
		err = fmt.Errorf("Client connect to Bitfinex websocket failed. err: [%v]", err)
		logger.Warnf("%v", err)
		return err
	}

	for _, v := range symbols.Values() {
		symbol := v.(string)
		logger.Infof("Subscribe book [%v].", symbol)

		// subscribe to BTCUSD book
		ctx, cxl1 := context.WithTimeout(context.Background(), time.Second*5)
		defer cxl1()
		_, err = client.SubscribeBook(ctx, symbol, bitfinex.Precision0, bitfinex.FrequencyRealtime, 25)
		if err != nil {
			err = fmt.Errorf("Client subscribe book [%v] failed. err: [%v]", symbol, err)
			logger.Warnf("%v", err)
			return err
		}
	}

	b.stopWg.Add(1)
	go func() {
		defer b.stopWg.Done()
		b.listenOrderBook(client)
	}()

	return nil
}

func (b *Bitfinex) listenOrderBook(client *websocket.Client) {

	objs := client.Listen()

	updateFn := func(update *bitfinex.BookUpdate) {

		logger.Debugf("Update order [%#v]", update)

		switch update.Symbol {
		case "tETHBTC":
			switch update.Side {
			case bitfinex.Bid:
				b.OrderBooks <- entity.OrderBook{
					Exchange: entity.Bitfinex,
					Time:     time.Now(),
					From:     entity.BTC,
					To:       entity.ETH,
					Price:    update.Price,
					Volume:   update.Amount,
				}
			case bitfinex.Ask:
				b.OrderBooks <- entity.OrderBook{
					Exchange: entity.Bitfinex,
					Time:     time.Now(),
					From:     entity.ETH,
					To:       entity.BTC,
					Price:    1 / update.Price,
					Volume:   update.Price * update.Amount,
				}

			default:
			}
		default:
			logger.Warnf("Unknown symbol [%v]", update.Symbol)
		}
	}

	for {
		select {
		case <-b.stop:
			logger.Infof("Stoping listen order books")
			return
		case obj := <-objs:
			switch obj.(type) {
			case error:
				logger.Warnf("Channel closed. err: [%v]", obj)
				return
			case *websocket.InfoEvent:
				info := obj.(*websocket.InfoEvent)
				logger.Infof("InfoEvent: [%#v]", info)
			case *websocket.SubscribeEvent:
				sub := obj.(*websocket.SubscribeEvent)
				logger.Infof("SubscribeEvent: [%#v]", sub)
			case *bitfinex.BookUpdateSnapshot:
				snapshot := obj.(*bitfinex.BookUpdateSnapshot)
				for _, update := range snapshot.Snapshot {
					updateFn(update)
				}
			case *bitfinex.BookUpdate:
				update := obj.(*bitfinex.BookUpdate)
				updateFn(update)
			default:
				logger.Warnf("Unknown type [%v]", reflect.TypeOf(obj))
				return
			}
		}
	}

}

func convertToSymbol(pair sets.Set) (string, bool) {

	if pair.Contains(entity.BTC, entity.ETH) {
		return bitfinex.TradingPrefix + bitfinex.ETHBTC, true
	} else {
		return "", false
	}
}
