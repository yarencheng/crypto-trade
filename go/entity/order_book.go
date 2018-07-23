package entity

import (
	"encoding/json"
	"time"
)

type OrderBook struct {
	Exchange Exchange
	Date     time.Time
	From     Currency
	To       Currency
	Price    float64
	Volume   float64
}

func (o OrderBook) String() string {
	j, _ := json.Marshal(o)
	return string(j)
}
