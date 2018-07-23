package entity

import (
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
