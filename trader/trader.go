package trader

import "github.com/yarencheng/crypto-trade/exchange"

type Trader interface {
	AddExchange(exchange *exchange.Exchange)
	Start()
	Stop()
}
