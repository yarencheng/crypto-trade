package data

import "github.com/yarencheng/crypto-trade/data/currencies"
import "encoding/json"

type Order struct {
	From currencies.Currency
	To   currencies.Currency
}

func (o Order) String() string {
	j, _ := json.Marshal(o)
	return string(j)
}
