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

	// create table
	if _, err := this.db.Exec(`
		CREATE TABLE orders (
			exchange TEXT NOT NULL,
			'from' TEXT NOT NULL,
			'to' TEXT NOT NULL,
			price REAL NOT NULL,
			volume REAL NOT NULL
		);`); err != nil {
		return fmt.Errorf("Create orders table failed: [%v]", err)
	}
	if _, err := this.db.Exec(`
		CREATE TABLE wallets (
			date DATETIME NOT NULL,
			exchange TEXT NOT NULL,
			currency TEXT NOT NULL,
			volume REAL NOT NULL
		);`); err != nil {
		return fmt.Errorf("Create wallets table failed: [%v]", err)
	}

	// create index
	if _, err := this.db.Exec("CREATE INDEX orders_exchange_idx ON orders (exchange);"); err != nil {
		return fmt.Errorf("Create orders_exchange_idx index failed: [%v]", err)
	}
	if _, err := this.db.Exec("CREATE INDEX orders_from_idx ON orders ('from');"); err != nil {
		return fmt.Errorf("Create orders_from_idx index failed: [%v]", err)
	}
	if _, err := this.db.Exec("CREATE INDEX orders_to_idx ON orders ('to');"); err != nil {
		return fmt.Errorf("Create orders_to_idx index failed: [%v]", err)
	}
	if _, err := this.db.Exec("CREATE INDEX orders_price_idx ON orders ('price');"); err != nil {
		return fmt.Errorf("Create orders_price_idx index failed: [%v]", err)
	}
	if _, err := this.db.Exec("CREATE UNIQUE INDEX orders_unique_idx ON orders (price, exchange, 'from', 'to');"); err != nil {
		return fmt.Errorf("Create orders_unique_idx index failed: [%v]", err)
	}

	// PRAGMA
	if _, err := this.db.Exec("PRAGMA auto_vacuum=1;"); err != nil {
		return fmt.Errorf("Set auto_vacuum failed: [%v]", err)
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

	r, err := this.db.Query("SELECT exchange, currency, SUM(volume) as volume FROM wallets GROUP BY exchange,currency;")
	if err != nil {
		return fmt.Errorf("Query failed. err: [%v]", err)
	}

	for r.Next() {
		var ex string
		var cur string
		var volume float64
		err = r.Scan(&ex, &cur, &volume)
		if err != nil {
			return fmt.Errorf("Scan failed. err: [%v]", err)
		}
		logger.Infof("%v %v:%v", ex, cur, volume)
	}

	return nil
}

func (this *SqliteReproducer) update(e *entity.OrderBookEvent) error {

	switch e.Type {
	case entity.ExchangeRestart:
		if _, err := this.db.Exec("DELETE FROM orders WHERE exchange == ?", e.Exchange); err != nil {
			return fmt.Errorf("Delete old order with exchange [%v] failed. err: [%v]", e.Exchange, err)
		}
		return nil
	case entity.Update:
	default:
		return fmt.Errorf("Unknown type [%v]", e.Type)
	}

	if e.Volume == 0 {
		if _, err := this.db.Exec(
			"DELETE FROM orders WHERE exchange == ? AND 'from' == ? AND 'to' == ? AND price == ?;",
			e.Exchange, e.From, e.To, e.Price,
		); err != nil {
			return fmt.Errorf("Delete all data from exchange [%v] failed, err: [%v]", e.Exchange, err)
		}
	} else {

		if _, err := this.db.Exec(
			"INSERT OR REPLACE INTO orders (exchange , 'from' , 'to' , price , volume) VALUES (?,?,?,?,?);",
			e.Exchange, e.From, e.To, e.Price, e.Volume,
		); err != nil {
			return err
		}
	}

	return nil
}

func (this *SqliteReproducer) buy(e *entity.BuyOrderEvent) error {
	switch e.Type {
	case entity.None:
		return nil
	case entity.FillOrKill:

	default:
		return fmt.Errorf("Unknown type [%v]", e.Type)
	}

	r, err := this.db.Query(`
		SELECT
			volume
		FROM orders
		WHERE exchange == ? AND 'from' == ? AND 'to' == ? AND price == ?`,
		e.Exchange, e.From, e.To, e.Price)
	if err != nil {
		return fmt.Errorf("Query order in stock failed. err: [%v]", err)
	}

	if !r.Next() {
		return fmt.Errorf("out of stock")
	}

	var volume float64
	err = r.Scan(&volume)
	if err != nil {
		return fmt.Errorf("Scan failed. err: [%v]", err)
	}

	if e.Volume > volume {
		return fmt.Errorf("out of stock")
	}

	if r.Next() {
		return fmt.Errorf("duplicated data")
	}

	if _, err := this.db.Exec(
		"INSERT INTO wallets (date, exchange, currency, volume) VALUES (?,?,?,?);",
		time.Now(), e.Exchange, e.To, e.Volume,
	); err != nil {
		return err
	}

	if _, err := this.db.Exec(
		"INSERT INTO wallets (date, exchange, currency, volume) VALUES (?,?,?,?);",
		time.Now(), e.Exchange, e.From, -(e.Price * e.Volume),
	); err != nil {
		return err
	}

	return nil
}
