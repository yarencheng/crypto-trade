package exchange

import (
	"github.com/yarencheng/crypto-trade/currency"
	"github.com/yarencheng/crypto-trade/order"
)

type Exchange interface {
	GetName() string
	Ping() float64
	GetOrders(from currency.Currency, to currency.Currency) chan order.Order
}
