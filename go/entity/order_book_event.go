package entity

import (
	"time"
)

type OrderBookEventType string

const (
	ExchangeRestart OrderBookEventType = "ExchangeRestart"
	Update          OrderBookEventType = "Update"
)

type OrderBookEvent struct {
	Type     OrderBookEventType
	Date     time.Time
	Exchange Exchange
	From     Currency
	To       Currency
	Price    float64
	Volume   float64
}
