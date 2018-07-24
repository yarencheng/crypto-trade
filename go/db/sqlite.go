package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

const sqliteSchema = `
-- table
CREATE TABLE IF NOT EXISTS orders (
	exchange 	TEXT 		NOT NULL,
	'from' 		TEXT 		NOT NULL,
	'to' 		TEXT 		NOT NULL,
	price 		REAL 		NOT NULL,
	volume 		REAL 		NOT NULL,
	update_date	datetime 	NOT NULL
);

-- index
CREATE 			INDEX IF NOT EXISTS orders_exchange_idx 	ON orders (exchange);
CREATE 			INDEX IF NOT EXISTS orders_from_idx 		ON orders ('from');
CREATE 			INDEX IF NOT EXISTS orders_to_idx 			ON orders ('to');
CREATE 			INDEX IF NOT EXISTS orders_price_idx 		ON orders ('price');
CREATE UNIQUE 	INDEX IF NOT EXISTS orders_unique_idx 		ON orders (price, exchange, 'from', 'to');

-- table
CREATE TABLE IF NOT EXISTS wallets (
	exchange 	TEXT 		NOT NULL,
	currency	TEXT 		NOT NULL,
	volume 		REAL 		NOT NULL,
	date		datetime 	NOT NULL
);
CREATE UNIQUE 	INDEX IF NOT EXISTS wallets_date_idx	ON wallets ('date');

-- table
CREATE TABLE IF NOT EXISTS order_book_events (
	date		datetime 	NOT NULL,
	'type'	 	TEXT 		NOT NULL,
	exchange 	TEXT 		NOT NULL,
	'from' 		TEXT 		NOT NULL,
	'to' 		TEXT 		NOT NULL,
	price 		REAL 		NOT NULL,
	volume 		REAL 		NOT NULL
);

CREATE 			INDEX IF NOT EXISTS order_book_events_exchange_idx 		ON order_book_events (exchange);
CREATE 			INDEX IF NOT EXISTS order_book_events_from_idx 			ON order_book_events ('from');
CREATE 			INDEX IF NOT EXISTS order_book_events_to_idx 			ON order_book_events ('to');
CREATE 			INDEX IF NOT EXISTS order_book_events_price_idx 		ON order_book_events ('price');
CREATE UNIQUE 	INDEX IF NOT EXISTS order_book_events_date_idx 			ON order_book_events ('date');

-- table
CREATE TABLE IF NOT EXISTS process_time (
	event_date	DATETIME 	NOT NULL,
	delayNs		INTEGER		NOT NULL
);
CREATE UNIQUE 	INDEX IF NOT EXISTS process_time_events_date_idx	ON process_time ('event_date');

PRAGMA auto_vacuum=1;
`

type DB struct {
	*sql.DB
}

func OpenSQLite(path string) (*DB, error) {
	this := &DB{}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("Open sqlite failed. err: [%v]", err)
	}

	_, err = db.Exec(sqliteSchema)
	if err != nil {
		return nil, fmt.Errorf("Create schema failed. err: [%v]", err)
	}

	this.DB = db

	return this, nil
}

func (this *DB) UpdateOrder(ob *entity.OrderBook) error {

	_, err := this.Exec(`
		INSERT OR REPLACE INTO orders
		('exchange', 'from', 'to', 'price', 'volume', 'update_date')
		VALUES (?, ?, ?, ?, ?, ?);`,
		ob.Exchange, ob.From, ob.To, ob.Price, ob.Volume, ob.Date,
	)

	if err != nil {
		return fmt.Errorf("Insert order [%v] failed. err: [%v]", ob, err)
	}

	return nil
}

func (this *DB) CountOrderBookEvent() (int64, error) {

	r, err := this.Query(`
		SELECT COUNT(*) FROM order_book_events;`,
	)
	if err != nil {
		return 0, fmt.Errorf("Query failed. err: [%v]", err)
	}

	var count int64

	if !r.Next() {
		return 0, fmt.Errorf("No result")
	}

	err = r.Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("Scan failed. err: [%v]", err)
	}

	return count, nil
}

func (this *DB) RemoveOrderByExchange(ex entity.Exchange) error {

	r, err := this.Exec(`DELETE FROM orders WHERE exchange = ?;`, ex)

	if err != nil {
		return fmt.Errorf("Delete all orders with exchange [%v] failed. err: [%v]", ex, err)
	}

	n, err := r.RowsAffected()
	if err != nil {
		return fmt.Errorf("Get number of deleted orders failed. err: [%v]", err)
	}

	logger.Debugf("Delete [%v] orders", n)

	return nil
}
