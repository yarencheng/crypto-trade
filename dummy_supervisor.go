package cryptotrade

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
)

type DummySupervisor struct {
	isRun bool
}

func NewDummySupervisor() *DummySupervisor {
	var d DummySupervisor
	d.isRun = false
	return &d
}

func (dummy *DummySupervisor) Supervise(orders chan data.Order) {

	dummy.isRun = true

	for {
		select {
		case order := <-orders:
			logrus.Infoln("receive order", order)
		default:
			if !dummy.isRun {
				break
			}
			time.Sleep(time.Second)
		}
	}
}

func (dummy *DummySupervisor) Stop() {
	dummy.isRun = false
}
