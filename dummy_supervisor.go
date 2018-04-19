package cryptotrade

import (
	"github.com/sirupsen/logrus"
	"github.com/yarencheng/crypto-trade/data"
)

type DummySupervisor struct {
}

func NewDummySupervisor() *DummySupervisor {
	var d DummySupervisor

	return &d
}

func (dummy *DummySupervisor) Supervise(orders chan data.Order) {

	for order := range orders {
		logrus.Infoln("receive order", order)
	}
}
