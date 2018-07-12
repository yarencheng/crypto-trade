package entity

import (
	"time"
)

type OrderBookEventType string

const (
	Update      OrderBookEventType = "Update"
	RemovePrice OrderBookEventType = "RemovePrice"
	RemoveAll   OrderBookEventType = "RemoRemoveAllvePrice"
)

type OrderBookEvent struct {
	Type   OrderBookEventType
	Date   time.Time
	From   Currency
	To     Currency
	Price  float64
	Volume float64
}
