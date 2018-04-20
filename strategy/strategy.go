package strategy

import "github.com/yarencheng/crypto-trade/data"

type Strategy interface {
	ReadOrders(chan data.Order)
	ReadTradeResults(chan data.TradeResult)
	WriteTrades() chan data.Trade
}
