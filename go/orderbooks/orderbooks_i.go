package orderbooks

import "github.com/yarencheng/crypto-trade/go/entity"

type OrderBooksI interface {
	Update(ob *entity.OrderBook) error
	RemoveExchange(ex entity.Exchange) error
}
