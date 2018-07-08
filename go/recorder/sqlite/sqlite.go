package sqlite

import (
	"context"
	"fmt"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/yarencheng/crypto-trade/go/entity"
	"github.com/yarencheng/crypto-trade/go/logger"
)

type Sqlite struct {
	Path       string
	OrderBooks <-chan entity.OrderBook
	db         *gorm.DB
	stop       chan int
	stopWg     sync.WaitGroup
}

func New() *Sqlite {
	sqlite := &Sqlite{
		stop: make(chan int, 1),
	}
	return sqlite
}

func (s *Sqlite) Start() error {

	logger.Infof("Starting")

	logger.Infof("Open [%v]", s.Path)
	db, err := gorm.Open("sqlite3", s.Path)
	if err != nil {
		err = fmt.Errorf("Failed to connect to [%v]. err: [%v]", s.Path, err)
		logger.Warnf("%v", err)
		return err
	}
	s.db = db

	logger.Infof("Started")

	// Migrate the schema
	db.AutoMigrate(&entity.OrderBook{})
	if s.db.Error != nil {
		err = fmt.Errorf("Failed to migrate order. err: [%v]", s.db.Error)
		logger.Warnf("%v", err)
		return err
	}

	s.stopWg.Add(1)
	go func() {
		s.recordOrder()
		s.stopWg.Done()
	}()

	logger.Infof("Started")

	return nil
}

func (s *Sqlite) Stop(ctx context.Context) error {
	logger.Info("Stopping")
	defer logger.Info("Stopped")

	wait := make(chan int, 1)

	go func() {
		close(s.stop)
		s.stopWg.Wait()
		s.db.Close()
		wait <- 1
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
		return nil
	}
}

func (s *Sqlite) recordOrder() {
	for {
		select {
		case <-s.stop:
			return
		case order := <-s.OrderBooks:
			logger.Debugf("Get order [%#v]", order)
			s.db.Create(&order)
			if s.db.Error != nil {
				logger.Warnf("Failed to insert order [%#v]. err: [%v]", order, s.db.Error)
				return
			}
		}
	}
}
