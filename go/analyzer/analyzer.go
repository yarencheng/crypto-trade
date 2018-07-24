package analyzer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/yarencheng/crypto-trade/go/db"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Analyzer struct {
	OrderBookPath string
	ResultPath    string
	stop          chan int
	wg            sync.WaitGroup
	OrderBooks    chan<- entity.OrderBookEvent
	BuyOrders     <-chan entity.BuyOrderEvent
	db            *db.DB
	orderBookDB   *db.DB
}

func New() *Analyzer {
	this := &Analyzer{
		stop: make(chan int, 1),
	}
	return this
}
func (this *Analyzer) Start() error {
	logger.Infoln("Starting")

	var err error
	this.db, err = db.OpenSQLite(this.ResultPath)
	if err != nil {
		return fmt.Errorf("Open sqlite failed. err: [%v]", err)
	}

	logger.Infof("Open [%v]", this.OrderBookPath)
	this.orderBookDB, err = db.OpenSQLite(this.OrderBookPath)
	if err != nil {
		return fmt.Errorf("Open sqlite failed. err: [%v]", err)
	}

	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		this.worker()
	}()

	logger.Infoln("Started")

	return nil
}

func (this *Analyzer) Stop(ctx context.Context) error {
	logger.Infoln("Stopping")

	wg := sync.WaitGroup{}
	wait := make(chan int)
	var err error

	wg.Add(1)
	go func() {
		close(this.stop)
		this.wg.Wait()
		defer close(wait)

		if e := this.db.Close(); e != nil {
			logger.Warnf("Close in-memory sqlite failed. err: [%v]", e)
			err = e
			return
		}
		if e := this.orderBookDB.Close(); e != nil {
			logger.Warnf("Close sqlite failed. err: [%v]", e)
			err = e
			return
		}
	}()

	select {
	case <-ctx.Done():
		logger.Warnf("Stop with an error [%v]", ctx.Err())
		return ctx.Err()
	case <-wait:
	}

	if err != nil {
		logger.Warnf("Stop with an error [%v]", err)
	} else {
		logger.Infoln("Stopped")
	}

	return err
}

func (this *Analyzer) worker() {
	logger.Infoln("Worker started")
	defer logger.Infoln("Worker finished")

	query, err := this.orderBookDB.Prepare(`
		SELECT
			order_book_events.'type',
			order_book_events.'exchange',
			order_book_events.'from',
			order_book_events.'to',
			order_book_events.'price',
			order_book_events.'volume',
			order_book_events.'date'
		FROM 		order_book_events
		WHERE 		order_book_events.'date' > ?
		ORDER BY	order_book_events.'date'
		LIMIT		100;
	`)
	if err != nil {
		logger.Warnf("Create prepare statement for querying order failed. err: [%v]", err)
		return
	}

	t := time.Unix(0, 0)
	var tvpe entity.OrderBookEventType
	var exchange entity.Exchange
	var from entity.Currency
	var to entity.Currency
	var price float64
	var volume float64

	count := new(int64)
	allCount, err := this.orderBookDB.CountOrderBookEvent()
	if err != nil {
		logger.Warnf("Get count of orders failed. err: [%v]", err)
		return
	}

	logger.Infof("Start to process [%v] orders.", allCount)

	stoplog := make(chan int, 1)
	defer close(stoplog)
	go func() {
		t := time.Tick(time.Second)
		for {
			select {
			case <-stoplog:
				return
			case <-t:
				c := atomic.LoadInt64(count)
				logger.Infof("Process [%v-%v%%] orders.", c, c*100/allCount)
			}
		}
	}()

	for {

		r, err := query.Query(t)
		if err != nil {
			logger.Warnf("Query failed. err: [%v]", err)
			return
		}

		isEmpty := true
		processTime := make([]struct {
			date  time.Time
			delay time.Duration
		}, 0, 100)

		for r.Next() {
			atomic.AddInt64(count, 1)
			isEmpty = false

			err = r.Scan(&tvpe, &exchange, &from, &to, &price, &volume, &t)

			if err != nil {
				logger.Warnf("Scan failed. err: [%v]", err)
				return
			}

			order := entity.OrderBookEvent{
				Type: tvpe,
				OrderBook: entity.OrderBook{
					Exchange: exchange,
					Date:     t,
					From:     from,
					To:       to,
					Price:    price,
					Volume:   volume,
				},
			}

			logger.Debugf("order [%v]", order)

			start := time.Now()

			select {
			case <-this.stop:
				return
			case this.OrderBooks <- order:
			}

			var buy entity.BuyOrderEvent
			select {
			case <-this.stop:
				return
			case buy = <-this.BuyOrders:
			}

			delay := time.Now().Sub(start)

			processTime = append(processTime, struct {
				date  time.Time
				delay time.Duration
			}{
				order.Date,
				delay,
			})

			logger.Debugf("buy [%v]", buy)

		}

		err = this.recordProcessTimes(processTime)
		if err != nil {
			logger.Warnf("Record process time failed. err: [%v]", err)
			return
		}

		r.Close()

		if isEmpty {
			break
		}
	}
}

func (this *Analyzer) recordProcessTimes(data []struct {
	date  time.Time
	delay time.Duration
}) error {

	tx, err := this.db.Begin()
	if err != nil {
		return fmt.Errorf("Start TX failed. err: [%v]", err)
	}

	// recordProcessTimeStmt, err := this.db.Prepare(`
	// 		INSERT INTO process_time ('event_date', 'delayNs')
	// 		VALUES (?, ?)
	// 	;`)

	// if err != nil {
	// 	return fmt.Errorf("Create prepare statement for querying order failed. err: [%v]", err)
	// }

	for _, d := range data {
		_, err := this.db.Exec(`
		INSERT INTO process_time ('event_date', 'delayNs')
		VALUES (?, ?)
	;`, d.date, d.delay)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("Update time failed. err: [%v]", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Commit TX failed. err: [%v]", err)
	}

	return nil

}

func (this *Analyzer) summary() error {

	// r, err := this.db.Query("SELECT exchange, currency, SUM(volume) as volume FROM wallets GROUP BY exchange,currency;")
	// if err != nil {
	// 	return fmt.Errorf("Query failed. err: [%v]", err)
	// }

	// for r.Next() {
	// 	var ex string
	// 	var cur string
	// 	var volume float64
	// 	err = r.Scan(&ex, &cur, &volume)
	// 	if err != nil {
	// 		return fmt.Errorf("Scan failed. err: [%v]", err)
	// 	}
	// 	logger.Infof("%v %v:%v", ex, cur, volume)
	// }

	return nil
}

func (this *Analyzer) update(e *entity.OrderBookEvent) error {

	// switch e.Type {
	// case entity.ExchangeRestart:
	// 	if _, err := this.db.Exec("DELETE FROM orders WHERE exchange == ?", e.Exchange); err != nil {
	// 		return fmt.Errorf("Delete old order with exchange [%v] failed. err: [%v]", e.Exchange, err)
	// 	}
	// 	return nil
	// case entity.Update:
	// default:
	// 	return fmt.Errorf("Unknown type [%v]", e.Type)
	// }

	// if e.Volume == 0 {
	// 	if _, err := this.db.Exec(
	// 		"DELETE FROM orders WHERE exchange == ? AND 'from' == ? AND 'to' == ? AND price == ?;",
	// 		e.Exchange, e.From, e.To, e.Price,
	// 	); err != nil {
	// 		return fmt.Errorf("Delete all data from exchange [%v] failed, err: [%v]", e.Exchange, err)
	// 	}
	// } else {

	// 	if _, err := this.db.Exec(
	// 		"INSERT OR REPLACE INTO orders (exchange , 'from' , 'to' , price , volume) VALUES (?,?,?,?,?);",
	// 		e.Exchange, e.From, e.To, e.Price, e.Volume,
	// 	); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (this *Analyzer) buy(e *entity.BuyOrderEvent) error {
	// switch e.Type {
	// case entity.None:
	// 	return nil
	// case entity.FillOrKill:

	// default:
	// 	return fmt.Errorf("Unknown type [%v]", e.Type)
	// }

	// r, err := this.db.Query(`
	// 	SELECT
	// 		volume
	// 	FROM orders
	// 	WHERE exchange == ? AND 'from' == ? AND 'to' == ? AND price == ?`,
	// 	e.Exchange, e.From, e.To, e.Price)
	// if err != nil {
	// 	return fmt.Errorf("Query order in stock failed. err: [%v]", err)
	// }

	// if !r.Next() {
	// 	return fmt.Errorf("out of stock")
	// }

	// var volume float64
	// err = r.Scan(&volume)
	// if err != nil {
	// 	return fmt.Errorf("Scan failed. err: [%v]", err)
	// }

	// if e.Volume > volume {
	// 	return fmt.Errorf("out of stock")
	// }

	// if r.Next() {
	// 	return fmt.Errorf("duplicated data")
	// }

	// if _, err := this.db.Exec(
	// 	"INSERT INTO wallets (date, exchange, currency, volume) VALUES (?,?,?,?);",
	// 	time.Now(), e.Exchange, e.To, e.Volume,
	// ); err != nil {
	// 	return err
	// }

	// if _, err := this.db.Exec(
	// 	"INSERT INTO wallets (date, exchange, currency, volume) VALUES (?,?,?,?);",
	// 	time.Now(), e.Exchange, e.From, -(e.Price * e.Volume),
	// ); err != nil {
	// 	return err
	// }

	return nil
}
