package exchanges

import (
	"github.com/yarencheng/crypto-trade/data"
)

type ExchangeI interface {
	GetName() string
	Ping() float64
	GetOrders(from data.Currency, to data.Currency) (<-chan data.Order, error)
	Buy(o data.Order) chan data.TradeResult
}
