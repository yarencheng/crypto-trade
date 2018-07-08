package entity

import (
	"encoding/json"
)

type OrderBook struct {
	Exchange Exchange
	From     Currency
	To       Currency
	Price    float64
	Volume   float64
}

func (o OrderBook) String() string {
	j, _ := json.Marshal(o)
	return string(j)
}
