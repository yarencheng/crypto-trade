package reproducer

import (
	"context"
	"reflect"
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
}

func New() *SqliteReproducer {
	this := &SqliteReproducer{
		stop: make(chan int, 1),
	}
	return this
}
func (this *SqliteReproducer) Start() error {
	logger.Infoln("Starting")

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

	cur := time.Unix(0, 0)

	for {
		var orders []entity.OrderBookEvent
		db.Where("date > ?", cur).Order("date").Limit(100).Find(&orders)

		if len(orders) == 0 {
			break
		}

		for _, order := range orders {
			logger.Warnf("order = %v", reflect.ValueOf(order))

			select {
			case <-this.stop:
				break
			case this.OrderBooks <- order:
			}

		}

		// break
	}

}
