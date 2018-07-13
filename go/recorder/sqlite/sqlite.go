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
	OrderBooks <-chan entity.OrderBookEvent
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
	db.AutoMigrate(&entity.OrderBookEvent{})
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

	tx := s.db.Begin()
	count := 0
	max := 1000

	for {
		select {
		case <-s.stop:
			if err := tx.Commit().Error; err != nil {
				logger.Errorf("Commit failed, err: [%v]", err)
				return
			}
			return
		case order := <-s.OrderBooks:
			logger.Debugf("Get order [%#v]", order)

			if err := tx.Create(&order).Error; err != nil {
				logger.Errorf("Create [%#v] failed, err: [%v]", order, err)
				tx.Rollback()
				return
			}
			count++

			if count == max {
				count = 0
				if err := tx.Commit().Error; err != nil {
					logger.Errorf("Commit failed, err: [%v]", err)
					return
				}
				tx = s.db.Begin()
			}
		}
	}
}
