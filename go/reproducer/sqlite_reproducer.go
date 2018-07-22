package reproducer

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type SqliteReproducer struct {
	Path       string
	stop       chan int
	wg         sync.WaitGroup
	OrderBooks chan<- entity.OrderBookEvent
	BuyOrders  <-chan entity.BuyOrderEvent
	db         *sql.DB
}

type Order struct {
	Exchange entity.Exchange
	From     entity.Currency
	To       entity.Currency
	Price    float64
	Volume   float64
}

type Wallet struct {
	Currency entity.Currency
	Volume   float64
}

func New() *SqliteReproducer {
	this := &SqliteReproducer{
		stop: make(chan int, 1),
	}
	return this
}
func (this *SqliteReproducer) Start() error {
	logger.Infoln("Starting")

	var err error
	this.db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("Open in-memory sqlite failed. err: [%v]", err)
	}
	if _, err := this.db.Exec("CREATE TABLE orders (exchang TEXT, from TEXT, to TEXT, price real, volume, REAL);"); err != nil {
		return fmt.Errorf("Create orders table failed: [%v]", err)
	}

	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		this.worker()
	}()

	logger.Infoln("Started")

	return nil
}

func (this *SqliteReproducer) Stop(ctx context.Context) error {
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

func (this *SqliteReproducer) worker() {
	logger.Infoln("Worker started")
	defer logger.Infoln("Worker finished")

	logger.Infof("Open [%v]", this.Path)
	db, err := gorm.Open("sqlite3", this.Path)
	if err != nil {
		logger.Warnf("Failed to connect to [%v]. err: [%v]", this.Path, err)
		return
	}

	var allCount, count, percent int64
	cur := time.Unix(0, 0)
	var allProcessTime time.Duration
	db.Model(&entity.OrderBookEvent{}).Count(&allCount)
Loop:
	for {
		var orders []entity.OrderBookEvent
		db.Where("date > ?", cur).Order("date").Limit(100).Find(&orders)

		if len(orders) == 0 {
			break
		}

		for _, order := range orders {
			count++
			if count*100/allCount >= percent+1 {
				percent = count * 100 / allCount
				logger.Infof("Process %v%% [%v/%v] avgTime=%v", percent, count, allCount, allProcessTime/time.Duration(count))
			}

			if err := this.update(&order); err != nil {
				logger.Errorf("Update order [%v] failed. err: [%v]", order, err)
				return
			}

			startTime := time.Now()

			select {
			case <-this.stop:
				break Loop
			case this.OrderBooks <- order:
				buy := <-this.BuyOrders

				processTime := time.Now().Sub(startTime)
				allProcessTime += processTime
				logger.Debugf("Took [%v] to process [%v]", processTime, order)

				if err := this.buy(&buy); err != nil {
					logger.Errorf("Update buy [%v] failed. err: [%v]", buy, err)
					return
				}
			}

		}

		cur = orders[len(orders)-1].Date
	}

	if err := this.summary(); err != nil {
		logger.Errorf("summary failed. err: [%v]", err)
		return
	}
}

func (this *SqliteReproducer) summary() error {

	// rows, err := this.liveOrder.
	// 	Table("wallets").
	// 	Select("sum(volume) as volume, currency").
	// 	Group("currency").
	// 	Rows()

	// if err != nil {
	// 	return fmt.Errorf("err: [%v]", err)
	// }

	// for rows.Next() {
	// 	var ex entity.Currency
	// 	var v float64
	// 	err := rows.Scan(&v, &ex)
	// 	if err != nil {
	// 		return fmt.Errorf("err: [%v]", err)
	// 	}
	// 	logger.Infof("%v:%v", ex, v)
	// }

	return nil
}

func (this *SqliteReproducer) update(e *entity.OrderBookEvent) error {

	// switch e.Type {
	// case entity.ExchangeRestart:
	// 	if err := this.liveOrder.Where("exchange == ?", e.Exchange).Delete(Order{}).Error; err != nil {
	// 		return fmt.Errorf("Delete old order with exchange [%v] failed. err: [%v]", e.Exchange, err)
	// 	}
	// 	return nil
	// case entity.Update:
	// default:
	// 	return fmt.Errorf("Unknown type [%v]", e.Type)
	// }

	// o := Order{
	// 	Exchange: e.Exchange,
	// 	From:     e.From,
	// 	To:       e.To,
	// 	Price:    e.Price,
	// 	Volume:   e.Volume,
	// }

	// if e.Volume == 0 {
	// 	if err := this.liveOrder.Where("price == ?", e.Price).Delete(Order{}).Error; err != nil {
	// 		return fmt.Errorf("Delete old order with price [%v] failed. err: [%v]", e.Price, err)
	// 	}
	// } else {

	// 	var o Order
	// 	if err := this.liveOrder.
	// 		Where(
	// 			"exchange == ? && from == ? && to == ? && price == ?", e.Exchange, e.From, e.To, e.Price,
	// 		).
	// 		Find(&o).Error; err != nil {
	// 		return fmt.Errorf("Delete old order with price [%v] failed. err: [%v]", e.Price, err)
	// 	}

	// 	o.Price = e.Price

	// 	if err := this.liveOrder.Save(&o).Error; err != nil {
	// 		return fmt.Errorf("Create order [%v] failed. err: [%v]", o, err)
	// 	}
	// }

	return nil
}

func (this *SqliteReproducer) buy(e *entity.BuyOrderEvent) error {
	// switch e.Type {
	// case entity.None:
	// 	return nil
	// case entity.FillOrKill:
	// 	add := &Wallet{
	// 		Currency: e.To,
	// 		Volume:   e.Volume,
	// 	}
	// 	sub := &Wallet{
	// 		Currency: e.From,
	// 		Volume:   -(e.Price * e.Volume),
	// 	}
	// 	logger.Infof("Add: %v  Sub: %v", add, sub)
	// 	if err := this.liveOrder.Create(sub).Error; err != nil {
	// 		return fmt.Errorf("Create Wallet failed. err: [%v]", err)
	// 	}
	// 	if err := this.liveOrder.Create(add).Error; err != nil {
	// 		return fmt.Errorf("Create Wallet failed. err: [%v]", err)
	// 	}
	// 	return nil
	// default:
	// 	return fmt.Errorf("Unknown type [%v]", e.Type)
	// }

	return nil
}
