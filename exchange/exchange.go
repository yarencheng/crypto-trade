package exchange

import "github.com/yarencheng/crypto-trade/data"

type Exchange interface {
	GetName() string
	WriteOrders() chan data.Order
	Ping() float64
}
