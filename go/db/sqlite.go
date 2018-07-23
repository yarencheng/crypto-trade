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
CREATE TABLE orders (
	exchange 	TEXT 		NOT NULL,
	'from' 		TEXT 		NOT NULL,
	'to' 		TEXT 		NOT NULL,
	price 		REAL 		NOT NULL,
	volume 		REAL 		NOT NULL,
	update_date	datetime 	NOT NULL
);

-- index
CREATE 			INDEX orders_exchange_idx 	ON orders (exchange);
CREATE 			INDEX orders_from_idx 		ON orders ('from');
CREATE 			INDEX orders_to_idx 		ON orders ('to');
CREATE 			INDEX orders_price_idx 		ON orders ('price');
CREATE UNIQUE 	INDEX orders_unique_idx 	ON orders (price, exchange, 'from', 'to');

PRAGMA auto_vacuum=1;
`

type DB struct {
	*sql.DB
}

func OpenSQLite() (*DB, error) {
	this := &DB{}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Open in-memory sqlite failed. err: [%v]", err)
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
