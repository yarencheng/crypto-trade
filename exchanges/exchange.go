package exchanges

import (
	"github.com/yarencheng/crypto-trade/data"
	"github.com/yarencheng/crypto-trade/data/currencies"
)

type Exchange interface {
	GetName() string
	Ping() float64
	GetOrders(from currencies.Currency, to currencies.Currency) (chan data.Order, error)
}
