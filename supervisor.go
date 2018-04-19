package cryptotrade

import "github.com/yarencheng/crypto-trade/data"

type Supervisor interface {
	Supervise(chan data.Order)
}
