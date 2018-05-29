package strategies

import (
	"github.com/yarencheng/crypto-trade/data"
)

type StrategyI interface {
	In(<-chan data.Order) error
	Out() (<-chan data.Order, error)
}
