package reproducer

import (
	"context"
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
	liveOrder  *gorm.DB
}

type Order struct {
	Exchange entity.Exchange `gorm:"index:exchange"`
	From     entity.Currency `gorm:"index:froms"`
	To       entity.Currency `gorm:"index:tos"`
	Price    float64         `gorm:"index:price"`
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
	this.liveOrder, err = gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("Open in-memory sqlite failed. err: [%v]", err)
	}
	if err := this.liveOrder.AutoMigrate(&Order{}).Error; err != nil {
		logger.Errorf("%v", err)
		return fmt.Errorf("Create table 'order' failed. err: [%v]", err)
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
				logger.Errorf("Update order [%v] failed. err: [%v]", err)
				return
			}

			startTime := time.Now()

			select {
			case <-this.stop:
				break Loop
			case this.OrderBooks <- order:
				<-this.BuyOrders

				processTime := time.Now().Sub(startTime)
				logger.Debugf("Took [%v] to process [%v]", processTime, order)

				allProcessTime += processTime
			}

		}

		cur = orders[len(orders)-1].Date
	}

}

func (this *SqliteReproducer) update(e *entity.OrderBookEvent) error {

	switch e.Type {
	case entity.ExchangeRestart:
		if err := this.liveOrder.Where("exchange == ?", e.Exchange).Delete(Order{}).Error; err != nil {
			return fmt.Errorf("Delete old order with exchange [%v] failed. err: [%v]", e.Exchange, err)
		}
		return nil
	case entity.Update:
	default:
		return fmt.Errorf("Unknown type [%v]", e.Type)
	}

	o := Order{
		Exchange: e.Exchange,
		From:     e.From,
		To:       e.To,
		Price:    e.Price,
		Volume:   e.Volume,
	}

	if o.Volume == 0 {
		if err := this.liveOrder.Where("price == ?", o.Price).Delete(Order{}).Error; err != nil {
			return fmt.Errorf("Delete old order with price [%v] failed. err: [%v]", o.Price, err)
		}
	} else {
		if err := this.liveOrder.Create(o).Error; err != nil {
			return fmt.Errorf("Create order [%v] failed. err: [%v]", o, err)
		}
	}

	return nil
}
