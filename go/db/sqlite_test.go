package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yarencheng/crypto-trade/go/entity"
)

func Test_SQLite_OpenSQLite(t *testing.T) {
	t.Parallel()

	// action
	db, err := OpenSQLite()

	// assert
	assert.NotNil(t, db)
	assert.NoError(t, err)
}

func Test_SQLite_RemoveOrderByExchange(t *testing.T) {
	t.Parallel()

	// arrange
	db, err := OpenSQLite()
	require.NoError(t, err)
	require.NotNil(t, db)

	_, err = db.Exec(`
		INSERT INTO orders ('exchange', 'from', 'to', 'price', 'volume', 'update_date')
		VALUES (?, ?, ?, ?, ?, ?);`,
		"eee", "fff", "ttt", 111, 222, time.Now(),
	)
	require.NoError(t, err)

	// action
	err = db.RemoveOrderByExchange(entity.Exchange("eee"))
	require.NoError(t, err)

	// assert
	r, err := db.Query(`SELECT * FROM orders WHERE 'exchange' == ?;`, "eee")
	require.NoError(t, err)
	assert.False(t, r.Next())
}

func Test_SQLite_UpdateOrder(t *testing.T) {
	t.Parallel()

	// arrange
	db, err := OpenSQLite()
	require.NoError(t, err)
	require.NotNil(t, db)
	expected := &entity.OrderBook{
		Date:     time.Now(),
		Exchange: entity.Exchange("eee"),
		From:     entity.Currency("fff"),
		To:       entity.Currency("ttt"),
		Price:    111,
		Volume:   222,
	}

	// action
	err = db.UpdateOrder(expected)
	require.NoError(t, err)

	// assert
	r, err := db.Query(`SELECT * FROM orders ;`, "eee")
	require.NoError(t, err)
	require.True(t, r.Next())
	var date time.Time
	var exchange string
	var from string
	var to string
	var price float64
	var volume float64
	err = r.Scan(&exchange, &from, &to, &price, &volume, &date)
	require.NoError(t, err)
	require.False(t, r.Next())
}
