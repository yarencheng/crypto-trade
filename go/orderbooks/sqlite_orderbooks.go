package orderbooks

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

const schema = `
-- table
CREATE TABLE orders (
	exchange 	TEXT NOT NULL,
	'from' 		TEXT NOT NULL,
	'to' 		TEXT NOT NULL,
	price 		REAL NOT NULL,
	volume 		REAL NOT NULL
);

-- index
CREATE 			INDEX orders_exchange_idx 	ON orders (exchange);
CREATE 			INDEX orders_from_idx 		ON orders ('from');
CREATE 			INDEX orders_to_idx 		ON orders ('to');
CREATE 			INDEX orders_price_idx 		ON orders ('price');
CREATE UNIQUE 	INDEX orders_unique_idx 	ON orders (price, exchange, 'from', 'to');

PRAGMA auto_vacuum=1;
`

type SqliteOrderBooks struct {
	db   *sql.DB
	stop chan int
	wg   sync.WaitGroup
}

func NewSqliteOrderBooks() (*SqliteOrderBooks, error) {
	this := &SqliteOrderBooks{
		stop: make(chan int, 1),
	}

	return this, nil
}

func (this *SqliteOrderBooks) Start(ctx context.Context) error {
	logger.Infoln("Starting")

	var err error
	this.db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		return fmt.Errorf("Open in-memory sqlite failed. err: [%v]", err)
	}

	_, err = this.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("Create schema failed. err: [%v]", err)
	}

	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		this.worker()
	}()

	logger.Infoln("Started")

	return nil
}

func (this *SqliteOrderBooks) Stop(ctx context.Context) error {
	logger.Infoln("Stopping")

	wg := sync.WaitGroup{}
	wait := make(chan int)
	var err error

	wg.Add(1)
	go func() {
		close(this.stop)
		this.wg.Wait()
		defer close(wait)

		if err = this.db.Close(); err != nil {
			logger.Warnf("Close sqlite connection failed. err: [%v]", err)
			return
		}
	}()

	select {
	case <-ctx.Done():
		logger.Warnf("Timeout to stop. err: [%v]", ctx.Err())
		return ctx.Err()
	case <-wait:
	}

	if err != nil {
		logger.Warnf("Stopped with an error [%v]", err)
		return err
	} else {
		logger.Infoln("Stopped")
		return nil
	}
}

func (this *SqliteOrderBooks) Update(ob *entity.OrderBook) error {
	return nil

}

func (this *SqliteOrderBooks) RemoveExchange(ex entity.Exchange) error {
	return nil
}

func (this *SqliteOrderBooks) worker() {

}
